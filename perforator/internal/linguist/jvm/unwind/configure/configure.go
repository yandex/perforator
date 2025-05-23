package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

func main() {
	err := mainImpl()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type compileCommand struct {
	Directory string `json:"directory"`
	File      string `json:"file"`
	Command   string `json:"command"`
}

type jdkInfo struct {
	defines    []string
	includes   []string
	sourceRoot string
}

func parseCompileCommands(ccJsonPath string) (jdkInfo, error) {
	ccJson, err := os.ReadFile(ccJsonPath)
	if err != nil {
		return jdkInfo{}, fmt.Errorf("failed to read compile_commands.json: %w", err)
	}
	var compileCommands []compileCommand
	err = json.Unmarshal(ccJson, &compileCommands)
	if err != nil {
		return jdkInfo{}, fmt.Errorf("failed to parse compile_commands.json: %w", err)
	}
	cmd := ""

	var sourceRoot string
	for _, cc := range compileCommands {
		if strings.Contains(cc.File, "share/oops/method.cpp") {
			if cmd != "" {
				return jdkInfo{}, fmt.Errorf("found more than file matching share/oops/method.cpp")
			}
			cmd = cc.Command
			sourceRoot = path.Join(cc.Directory, "../..")
		}
	}
	flags := strings.Split(cmd, " ")
	if len(flags) == 0 {
		return jdkInfo{}, fmt.Errorf("unexpected empty command: %q, cmd", cmd)
	}
	flags = flags[1:]
	var defines []string
	var includes []string
	for _, f := range flags {
		if strings.HasPrefix(f, "-I") {
			fmt.Fprintf(os.Stderr, "Processing include: %q\n", f)
			includes = append(includes, strings.TrimPrefix(f, "-I"))
		} else if strings.HasPrefix(f, "-D") {
			fmt.Fprintf(os.Stderr, "Processing define: %q\n", f)
			defines = append(defines, strings.TrimPrefix(f, "-D"))
		} else {
			fmt.Fprintf(os.Stderr, "Skipping unknown flag: %q\n", f)
		}
	}

	return jdkInfo{
		defines:    defines,
		includes:   includes,
		sourceRoot: sourceRoot,
	}, nil
}

type writerWrapper struct {
	w   io.Writer
	err error
}

func (w *writerWrapper) Write(p []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err = w.w.Write(p)
	if err != nil {
		w.err = err
	}
	return n, err
}

func writeYmakeInc(w io.Writer, jdkInfo jdkInfo) error {
	wr := &writerWrapper{w: w}
	_, _ = fmt.Fprintf(wr, "CFLAGS(\n")

	for _, include := range jdkInfo.includes {
		_, _ = fmt.Fprintf(wr, "    -I%s\n", include)
	}
	for _, define := range jdkInfo.defines {
		_, _ = fmt.Fprintf(wr, "    -D%s\n", define)
	}
	_, _ = fmt.Fprintf(wr, ")\n")
	return wr.err
}

var headers = []string{
	"src/hotspot/share/runtime/frame.hpp",
	"src/hotspot/share/code/codeCache.hpp",
	"src/hotspot/share/runtime/vmStructs.hpp",
}

func writeHeader(w io.Writer, jdkInfo jdkInfo) error {
	wr := &writerWrapper{w: w}

	for _, header := range headers {
		_, _ = fmt.Fprintf(wr, "#include \"%s/%s\"\n", jdkInfo.sourceRoot, header)
	}
	return wr.err
}

func mainImpl() error {
	ccJsonPath := flag.String("compile-commands-path", "", "Path to compile_commands.json")
	outDir := flag.String("out-dir", "", "Path to output directory")
	flag.Parse()
	if *ccJsonPath == "" {
		return fmt.Errorf("--compile-commands-path is required")
	}
	if *outDir == "" {
		return fmt.Errorf("--out-dir is required")
	}

	jdkInfo, err := parseCompileCommands(*ccJsonPath)
	if err != nil {
		return fmt.Errorf("failed to process compile_commands.json: %w", err)
	}

	ymakeInc, err := os.Create(path.Join(*outDir, "gen.inc"))
	if err != nil {
		return fmt.Errorf("failed to create gen.inc: %w", err)
	}

	err = writeYmakeInc(ymakeInc, jdkInfo)
	if err != nil {
		return fmt.Errorf("failed to write gen.inc: %w", err)
	}

	header, err := os.Create(path.Join(*outDir, "gen.h"))
	if err != nil {
		return fmt.Errorf("failed to create gen.h: %w", err)
	}

	err = writeHeader(header, jdkInfo)
	if err != nil {
		return fmt.Errorf("failed to write gen.h: %w", err)
	}
	return nil
}
