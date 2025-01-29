package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func unwrap[T any](value T, err error) T {
	check(err)
	return value
}

func main() {
	loop := flag.Bool("loop", false, "Spin forever")
	format := flag.String("format", "collapsed", "Profile format: pprof or collapsed")
	maxdepth := flag.Int("maxdepth", 128, "Truncate stacks that are taller than the limit")
	minweight := flag.Float64("minweight", 0.0001, "Truncate stacks that are narrower than the limit")
	flag.Parse()

	go func() {
		check(http.ListenAndServe("localhost:17851", nil))
	}()

	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	defer func() { _ = w.Flush() }()

	raw := unwrap(io.ReadAll(r))
	for {
		start := time.Now()
		fg := render.NewFlameGraph()
		fg.SetDepthLimit(*maxdepth)
		fg.SetMinWeight(*minweight)

		switch *format {
		case "collapsed":
			profile := unwrap(collapsed.Unmarshal(raw))
			check(fg.RenderCollapsed(profile, w))
		case "pprof":
			profile := unwrap(profile.ParseData(raw))
			check(fg.RenderPProf(profile, w))
		}

		if !*loop {
			break
		}
		fmt.Fprintf(os.Stderr, "Built flamegraph in %s\n", time.Since(start))
	}
}
