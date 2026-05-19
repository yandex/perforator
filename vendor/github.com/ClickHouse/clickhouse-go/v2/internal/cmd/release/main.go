package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var skipWorkingTreeIsDirtyCheck = flag.Bool("skip-working-tree-is-dirty-check", false, "Skip working tree is dirty check")

func main() {
	flag.Parse()

	if !(*skipWorkingTreeIsDirtyCheck) && gitRepositoryWorkingTreeIsDirty() {
		log.Fatalln("Git working tree is dirty")
		return
	}

	releaseURL, err := getLatestDraftReleaseURL()
	if err != nil {
		log.Fatalln(err)
		return
	}

	log.Println("Latest draft release URL:")
	log.Println(releaseURL)

	r, err := getRelease(releaseURL)
	if err != nil {
		log.Fatalln(err)
		return
	}

	log.Println("Release tag:")
	log.Println(r.TagName)
	log.Println("Release body:")
	log.Println(r.Body)

	major, minor, patch, err := parseSemVer(r.TagName)
	if err != nil {
		log.Fatalln(err)
		return
	}

	if len(r.Body) == 0 {
		log.Fatalln("Release body is empty")
		return
	}

	changelogPath := changelogFilePath()
	if err := prependReleaseToChangelog(changelogPath, r); err != nil {
		log.Fatalln(err)
		return
	}

	if err := updateClientInfo(major, minor, patch); err != nil {
		log.Fatalln(err)
		return
	}

	runGoGenerate()
	runGoFmt()

	if err := gitHubOutputReleaseURLIfAvailable(releaseURL); err != nil {
		log.Fatalln(err)
		return
	}
}

func gitHubOutputReleaseURLIfAvailable(url string) error {
	if len(url) == 0 {
		return nil
	}

	gitHubOutputFile, exists := os.LookupEnv("GITHUB_OUTPUT")
	if !exists {
		return nil
	}

	f, err := os.OpenFile(gitHubOutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := fmt.Fprintf(f, "RELEASE_URL=%s\n", url); err != nil {
		return err
	}

	return nil
}

func runGoFmt() {
	cmd := exec.Command("go", "fmt", "./...")
	cmd.Dir = getRootPath()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
		return
	}
}

func runGoGenerate() {
	cmd := exec.Command("go", "generate", "./...")
	cmd.Dir = getRootPath()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
		return
	}
}

func gitRepositoryWorkingTreeIsDirty() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		log.Fatalln(err)
		return false
	}
	return len(out) > 0
}

func parseSemVer(version string) (major, minor, patch int, err error) {
	// Define a regular expression to match SemVer format
	re := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

	// Apply the regular expression to the version string
	match := re.FindStringSubmatch(version)

	// Check if the version string matches the SemVer format
	if len(match) != 4 {
		err = fmt.Errorf("invalid SemVer format: %s", version)
		return
	}

	// Parse the major, minor, and patch components as integers
	major, err = strconv.Atoi(match[1])
	if err != nil {
		return
	}

	minor, err = strconv.Atoi(match[2])
	if err != nil {
		return
	}

	patch, err = strconv.Atoi(match[3])
	if err != nil {
		return
	}

	return
}

func prependReleaseToChangelog(changelogPath string, r release) error {
	f, err := os.OpenFile(changelogPath, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	f.Seek(0, io.SeekStart)
	fmt.Fprintf(f, "# %s, %s ", r.TagName, time.Now().Format("2006-01-02"))
	f.WriteString(r.Body)
	f.WriteString("\n\n")
	f.Write(content)

	return nil
}

func changelogFilePath() string {
	rootPath := getRootPath()
	changelogPath := rootPath + "/CHANGELOG.md"
	return changelogPath
}

func getRootPath() string {
	wd, _ := os.Getwd()
	rootPath := strings.Replace(wd, "internal/cmd/release", "", 1)
	return rootPath
}

func getRelease(releaseURL string) (release, error) {
	req, err := http.NewRequest(http.MethodGet, releaseURL, nil)
	if err != nil {
		return release{}, err
	}
	req.Header.Set("Authorization", "token "+os.Getenv("GITHUB_TOKEN"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return release{}, err
	}
	defer res.Body.Close()

	var rel release
	if err := json.NewDecoder(res.Body).Decode(&rel); err != nil {
		return release{}, err
	}

	return rel, nil
}

type release struct {
	URL     string `json:"url"`
	Body    string `json:"body"`
	TagName string `json:"tag_name"`
}

func getLatestDraftReleaseURL() (string, error) {
	// Fetch the latest release from GitHub repository using GitHub API
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/clickhouse/clickhouse-go/releases?per_page=100", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "token "+os.Getenv("GITHUB_TOKEN"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var releases []struct {
		URL   string `json:"url"`
		Draft bool   `json:"draft"`
	}
	if err := json.NewDecoder(res.Body).Decode(&releases); err != nil {
		return "", err
	}

	// filter out releases that are not drafts
	for i := 0; i < len(releases); i++ {
		if releases[i].Draft {
			return releases[i].URL, nil
		}
	}

	return "", fmt.Errorf("no draft releases found")
}

func updateClientInfo(major, minor, patch int) error {
	// Open the client_info.go file for reading and writing
	file, err := os.OpenFile(getRootPath()+"/client_info.go", os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the contents of the file into memory
	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Replace the ClientVersionMajor, ClientVersionMinor, and ClientVersionPatch lines
	reMajor := regexp.MustCompile(`ClientVersionMajor\s+=\s+\d+`)
	reMinor := regexp.MustCompile(`ClientVersionMinor\s+=\s+\d+`)
	rePatch := regexp.MustCompile(`ClientVersionPatch\s+=\s+\d+`)
	newLines := []string{
		fmt.Sprintf("ClientVersionMajor       = %d", major),
		fmt.Sprintf("ClientVersionMinor       = %d", minor),
		fmt.Sprintf("ClientVersionPatch       = %d", patch),
	}
	scanner := bufio.NewScanner(strings.NewReader(string(bytes)))
	var newContent string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case reMajor.MatchString(line):
			line = newLines[0]
		case reMinor.MatchString(line):
			line = newLines[1]
		case rePatch.MatchString(line):
			line = newLines[2]
		}
		newContent += line + "\n"
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Write the updated content back to the file
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Write([]byte(newContent)); err != nil {
		return err
	}

	return nil
}
