package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render"
)

func main() {
	var (
		lineNumbers = flag.Bool("line-numbers", true, "Show line numbers")
		fileNames   = flag.Bool("file-names", true, "Show file names")
		addrPolicy  = flag.String("addr-policy", "never", "Address render policy: never, unsymbolized, always")
		profilePath = flag.String("file", "", "Profile file path")
		maxSamples  = flag.Int("max-samples", 0, "Maximum number of samples to render (0 and less means no limit)")
	)
	flag.Parse()

	filePath := *profilePath
	if filePath == "" {
		args := flag.Args()
		if len(args) < 1 {
			fmt.Println("Error: Missing profile file path")
			fmt.Println("Usage: <this_executable> [flags] <pprof-file>")
			flag.PrintDefaults()
			os.Exit(1)
		}
		filePath = args[0]
	}

	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening profile file %s: %v\n", filePath, err)
		os.Exit(1)
	}
	defer f.Close()

	p, err := profile.Parse(f)
	if err != nil {
		fmt.Printf("Error parsing profile: %v\n", err)
		os.Exit(1)
	}

	text := render.NewTextFormatRenderer()
	text.SetLineNumbers(*lineNumbers)
	text.SetFileNames(*fileNames)
	text.SetMaxSamples(*maxSamples)

	switch *addrPolicy {
	case "unsymbolized":
		text.SetAddressRenderPolicy(render.RenderAddressesUnsymbolized)
	case "always":
		text.SetAddressRenderPolicy(render.RenderAddressesAlways)
	case "never":
		text.SetAddressRenderPolicy(render.RenderAddressesNever)
	default:
		fmt.Printf("Warning: Unknown address policy '%s', using default\n", *addrPolicy)
	}

	if err := text.AddProfile(p); err != nil {
		fmt.Printf("Error adding profile: %v\n", err)
		os.Exit(1)
	}

	data, err := text.RenderBytes()
	if err != nil {
		fmt.Printf("Error rendering profile: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s", data)
}
