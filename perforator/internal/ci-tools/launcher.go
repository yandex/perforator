package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"
)

func main() {
	err := mainImpl(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type config struct {
	archivePath   string
	scriptPath    string
	outPath       string
	remoteOutPath string
	token         string
	vmControlUrl  string
	pollVMUrl     string
	runID         string
	repoID        string
	localWorkDir  string
	envFile       string
	secretFiles   string
}

func getEnv(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return v, nil
}

func loadConfig() (*config, error) {
	var c config
	var err error
	c.archivePath, err = getEnv("ARCHIVE_PATH")
	if err != nil {
		return nil, err
	}
	c.scriptPath, err = getEnv("SCRIPT_PATH")
	if err != nil {
		return nil, err
	}
	c.outPath, err = getEnv("OUT_PATH")
	if err != nil {
		return nil, err
	}
	c.remoteOutPath, err = getEnv("REMOTE_OUT_PATH")
	if err != nil {
		return nil, err
	}
	c.token, err = getEnv("IAM_TOKEN")
	if err != nil {
		return nil, err
	}
	c.vmControlUrl, err = getEnv("CREATE_VM_URL")
	if err != nil {
		return nil, err
	}
	c.pollVMUrl, err = getEnv("POLL_VM_URL")
	if err != nil {
		return nil, err
	}
	c.runID, err = getEnv("RUN_ID")
	if err != nil {
		return nil, err
	}
	c.repoID, err = getEnv("REPO_ID")
	if err != nil {
		return nil, err
	}
	c.localWorkDir, err = getEnv("LOCAL_WORK_DIR")
	if err != nil {
		return nil, err
	}
	c.envFile, err = getEnv("ENV_FILE")
	if err != nil {
		return nil, err
	}
	c.secretFiles, err = getEnv("SECRET_FILES")
	if err != nil {
		return nil, err
	}
	return &c, nil
}

type retryableError struct {
	error
}

func callFunc[T any, U any](ctx context.Context, url string, request T, token string) (U, error) {
	var response U
	reqBody, err := json.Marshal(request)
	if err != nil {
		return response, fmt.Errorf("failed to serialize request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return response, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Add("Authorization", "Bearer "+token)
	httpReq.Header.Add("Content-Type", "application/json")
	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return response, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpRes.Body.Close()
	data, err := io.ReadAll(httpRes.Body)
	// intentionally happens before err check; this way message will likely be more helpful
	if httpRes.StatusCode >= 300 {
		err = fmt.Errorf("request failed with status code %d: %s", httpRes.StatusCode, string(data))
		if httpRes.StatusCode == 504 {
			err = &retryableError{err}
		}
		return response, err
	}
	if err != nil {
		return response, fmt.Errorf("failed to read response: %w", err)
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return response, fmt.Errorf("failed to deserialize response: %w", err)
	}
	return response, nil
}

type createVMRequest struct {
	Kind    string   `json:"kind"`
	RunID   string   `json:"run_id"`
	RepoID  string   `json:"repo_id"`
	PubKeys []string `json:"ssh_keys"`
}
type createVMResponse struct {
	BuildID     string `json:"build_id"`
	ExecutionID string `json:"execution_id"`
}

type deleteVMRequest struct {
	Kind    string `json:"kind"`
	BuildID string `json:"build_id"`
}

type deleteVMResponse struct {
	Status string `json:"status"`
}

type pollVMRequest struct {
	BuildID string `json:"build_id"`
}

type pollVMResponse struct {
	Status  string `json:"status"`
	Address string `json:"address"`
}

func validateAddress(address string) error {
	parts := strings.Split(address, ".")
	if len(parts) != 4 {
		return fmt.Errorf("address does not have 4 parts")
	}
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("address part is empty")
		}
		for _, d := range part {
			if d < '0' || d > '9' {
				return fmt.Errorf("address part contains non-digit character")
			}
		}
	}
	return nil
}

func keyGen(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "ssh-keygen", "-t", "rsa", "-b", "4096", "-f", path, "-N", "")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func loadEnv(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}
	env := make(map[string]string)
	for lineIdx, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)

		if len(parts) != 2 {
			return nil, fmt.Errorf("failed to parse env file line %d: %w", lineIdx, err)
		}
		env[parts[0]] = parts[1]
	}
	return env, nil
}

//go:embed wrapper.tmpl.sh
var wrapperTemplate string

type secret struct {
	Name  string
	Value string
}

type variable struct {
	Name  string
	Value string
}

type wrapperTemplateContext struct {
	Secrets   []secret
	Variables []variable
}

func makeWrapper(env map[string]string) (string, error) {
	var ctx wrapperTemplateContext
	for k, v := range env {
		ctx.Variables = append(ctx.Variables, variable{Name: k, Value: v})
	}

	templ := template.New("wrapper.sh")
	_, err := templ.Parse(wrapperTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to create template: %w", err)
	}
	buf := bytes.NewBuffer(nil)
	err = templ.Execute(buf, ctx)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}

func mainImpl(ctx context.Context) error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.InfoContext(ctx, "Loading config")
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	logger.InfoContext(ctx, "Creating job wrapper")
	jobEnv, err := loadEnv(config.envFile)
	if err != nil {
		return fmt.Errorf("failed to load job environment file: %w", err)
	}
	wrapperScript, err := makeWrapper(jobEnv)
	if err != nil {
		return fmt.Errorf("failed to create wrapper script: %w", err)
	}
	wrapperScriptPath := path.Join(config.localWorkDir, "wrapper.sh")
	err = os.WriteFile(wrapperScriptPath, []byte(wrapperScript), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}
	logger.InfoContext(ctx, "Generating SSH key")
	keyPath := path.Join(config.localWorkDir, "id_rsa")
	err = keyGen(ctx, keyPath)
	if err != nil {
		return fmt.Errorf("failed to generate SSH key: %w", err)
	}
	pubKey, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	logger.InfoContext(ctx, "Requesting VM")
	var createVMReq createVMRequest
	createVMReq.Kind = "create"
	createVMReq.RunID = config.runID
	createVMReq.RepoID = config.repoID
	createVMReq.PubKeys = []string{string(pubKey)}
	var createVMRes createVMResponse
	createVMRes, err = callFunc[createVMRequest, createVMResponse](ctx, config.vmControlUrl, createVMReq, config.token)
	if err != nil {
		return fmt.Errorf("failed to request VM: %w", err)
	}
	logger.InfoContext(ctx, "Requested VM", "build_id", createVMRes.BuildID, "execution_id", createVMRes.ExecutionID)
	logger.InfoContext(ctx, "Waiting for VM creation")
	var vmAddress string
	pollCtx, cancelPollCtx := context.WithTimeoutCause(ctx, 5*time.Minute, fmt.Errorf("VM creation timeout (5m) exceeded"))
	defer cancelPollCtx()
	for pollCtx.Err() == nil {
		var pollVMRes pollVMResponse
		pollVMRes, err = callFunc[pollVMRequest, pollVMResponse](pollCtx, config.pollVMUrl, pollVMRequest{BuildID: createVMRes.BuildID}, config.token)
		if err != nil {
			var rerr *retryableError
			if errors.As(err, &rerr) {
				logger.WarnContext(ctx, "Transient error", "error", err)
				time.Sleep(5 * time.Second)
				continue
			}
			return fmt.Errorf("failed to poll VM: %w", err)
		}
		logger.InfoContext(pollCtx, "Polled VM status", "status", pollVMRes.Status)
		if pollVMRes.Status == "CREATING" || pollVMRes.Status == "NOT_FOUND" {
			time.Sleep(10 * time.Second)
			continue
		}
		if pollVMRes.Status == "ERROR" {
			return fmt.Errorf("VM creation failed")
		}
		if pollVMRes.Status == "READY" {
			vmAddress = pollVMRes.Address
			break
		}
		return fmt.Errorf("unexpected VM status %q", pollVMRes.Status)
	}
	defer func() {
		logger.InfoContext(ctx, "Cleaning up")
		deleteReq := deleteVMRequest{
			Kind:    "delete",
			BuildID: createVMRes.BuildID,
		}
		deleteRes, err := callFunc[deleteVMRequest, deleteVMResponse](ctx, config.vmControlUrl, deleteReq, config.token)
		if err != nil {
			logger.WarnContext(ctx, "Failed to delete VM", "error", err)
		} else if deleteRes.Status != "OK" {
			logger.WarnContext(ctx, "Failed to delete VM", "response", deleteRes)
		}
	}()
	if pollCtx.Err() != nil {
		return fmt.Errorf("VM creation canceled: %w", pollCtx.Err())
	}
	cancelPollCtx()
	err = validateAddress(vmAddress)
	if err != nil {
		return fmt.Errorf("invalid VM address %q: %w", vmAddress, err)
	}
	logger.InfoContext(ctx, "VM created", "address", vmAddress)

	logger.InfoContext(ctx, "Transferring build context")
	cmds := []sftpCommand{
		{
			progress: &sftpCommandProgess{},
		},
		{
			upload: &sftpCommandUpload{
				src: config.archivePath,
				dst: "/home/builder/code.tgz",
			},
		},
		{
			upload: &sftpCommandUpload{
				src: wrapperScriptPath,
				dst: "/home/builder/wrapper.sh",
			},
		},
		{
			upload: &sftpCommandUpload{
				src: config.scriptPath,
				dst: "/home/builder/job.sh",
			},
		},
	}
	secretFiles := strings.Split(config.secretFiles, ",")
	for _, secretFile := range secretFiles {
		if secretFile == "" {
			continue
		}
		parts := strings.Split(secretFile, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid secret file transfer command %q", secretFile)
		}
		cmds = append(cmds, sftpCommand{
			upload: &sftpCommandUpload{
				src: parts[0],
				dst: parts[1],
			},
		})
	}
	sshOpts := sshSessionOptions{
		server:   vmAddress,
		username: "builder",
		identity: keyPath,
		extraOpts: []string{
			fmt.Sprintf("UserKnownHostsFile=%s/hosts", config.localWorkDir),
		},
	}
	ok := false
	for i := 0; i < 6; i++ {
		connOpts := sshOpts
		connOpts.extraOpts = append(connOpts.extraOpts[:], ("StrictHostKeyChecking=no"))
		connOpts.dir = "/home/builder"
		if i > 3 {
			connOpts.verbose = true
		}
		err = runSFTP(ctx, connOpts, cmds)
		if err != nil {
			logger.WarnContext(ctx, "Failed to transfer build context", "error", err, "attempt", i)
			time.Sleep(40 * time.Second)
		} else {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("failed to transfer build context: retry attempts exhausted")
	}
	logger.InfoContext(ctx, "Running build script")
	ok = false
	for i := 0; i < 5; i++ {
		err = runSSH(ctx, sshOpts, []string{"bash /home/builder/wrapper.sh"})
		var target transientSSHError
		if err == nil {
			ok = true
			break
		}
		if errors.Is(err, &target) {
			logger.WarnContext(ctx, "Failed to run build script", "error", err, "attempt", i)
			time.Sleep(15 * time.Second)
			continue
		}
		return fmt.Errorf("failed to run build script: %w", err)
	}
	if !ok {
		return fmt.Errorf("failed to run build script: retry attempts exhausted")
	}
	logger.InfoContext(ctx, "Downloading build artifacts")
	ok = false
	for i := 0; i < 5; i++ {
		connOpts := sshOpts
		connOpts.extraOpts = append(connOpts.extraOpts[:], ("StrictHostKeyChecking=no"))
		connOpts.dir = "/home/builder"
		if i > 3 {
			connOpts.verbose = true
		}
		err = runSFTP(ctx, connOpts, []sftpCommand{
			{
				download: &sftpCommandDownload{
					src: "/home/builder/job-exit-code",
					dst: fmt.Sprintf("%s/job-exit-code", config.localWorkDir),
				},
			},
			{
				download: &sftpCommandDownload{
					src: config.remoteOutPath,
					dst: config.outPath,
				},
			},
		})
		if err != nil {
			logger.WarnContext(ctx, "Failed to download build artifacts", "error", err, "attempt", i)
			time.Sleep(40 * time.Second)
		} else {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("failed to download build artifacts: retry attempts exhausted")
	}

	exitCodeS, err := os.ReadFile(fmt.Sprintf("%s/job-exit-code", config.localWorkDir))
	if err != nil {
		return fmt.Errorf("failed to read job exit code: %w", err)
	}
	exitCode, err := strconv.Atoi(strings.TrimSpace(string(exitCodeS)))
	if err != nil {
		return fmt.Errorf("failed to parse job exit code %q: %w", exitCodeS, err)
	}
	if exitCode != 0 {
		return fmt.Errorf("job exited with code %d", exitCode)
	}

	return nil

}
