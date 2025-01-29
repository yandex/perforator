package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/pprof/driver"
	"github.com/google/pprof/profile"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/atomicfs"
)

////////////////////////////////////////////////////////////////////////////////

type ProfileSink interface {
	Store(profile []byte) error
}

////////////////////////////////////////////////////////////////////////////////

func MakePProfSink(log log.Logger, address string, browser bool) (ProfileSink, error) {
	return &PProfProfileSink{log, address, browser}, nil
}

func MakeHTTPSink(log log.Logger, address string, browser bool) (ProfileSink, error) {
	return &HTTPProfileSink{log: log, bindAddress: address, wantBrowser: browser}, nil
}

func MakeFileSink(log log.Logger, path string) (ProfileSink, error) {
	return &FileProfileSink{log, path}, nil
}

////////////////////////////////////////////////////////////////////////////////

type FileProfileSink struct {
	log  log.Logger
	path string
}

func (s *FileProfileSink) Store(profile []byte) error {
	s.log.Info("Writing profile",
		log.String("path", s.path),
		log.Int("bytes", len(profile)),
	)

	output, err := atomicfs.Create(s.path)
	if err != nil {
		return err
	}
	defer output.Discard()

	_, err = output.Write(profile)
	if err != nil {
		return err
	}

	err = output.Close()
	if err != nil {
		return err
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type HTTPProfileSink struct {
	log               log.Logger
	bindAddress       string
	resolvableAddress string
	wantBrowser       bool
}

func (s *HTTPProfileSink) openInBrowser() error {
	if !s.wantBrowser {
		return nil
	}

	browserVariants := []string{"xdg-open", "open"}
	if browser, ok := os.LookupEnv("BROWSER"); ok {
		browserVariants = append(browserVariants, browser)
	}

	var errs []error
	for _, browser := range browserVariants {
		s.log.Info("Trying to open browser",
			log.String("binary", browser),
			log.String("address", s.resolvableAddress),
		)
		cmd := exec.Command(browser, s.resolvableAddress)

		err := cmd.Start()
		if err != nil {
			if !errors.Is(err, exec.ErrNotFound) {
				errs = append(errs, err)
			}
			continue
		}

		err = cmd.Wait()
		if err != nil {
			s.log.Warn("Failed to open browser", log.Error(err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to open browser: %w", errors.Join(errs...))
	} else {
		return fmt.Errorf("failed to open browser: no valid browser found")
	}
}

func (s *HTTPProfileSink) Store(profile []byte) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.log.Info("Got request", log.Any("url", r.URL))
		_, _ = w.Write(profile)
	})

	srv := http.Server{Handler: mux}
	ln, err := net.Listen("tcp", s.bindAddress)
	if err != nil {
		return err
	}

	addr := ln.Addr().String()
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}

	hostname, ok := getResolvableSelfHostname()
	if ok {
		addr = strings.ReplaceAll(addr, "[::]", hostname)
	}
	s.resolvableAddress = addr

	s.log.Info("Starting http server", log.String("address", s.resolvableAddress))

	var g errgroup.Group
	g.Go(func() error {
		return srv.Serve(ln)
	})
	if s.wantBrowser {
		g.Go(s.openInBrowser)
	}
	return g.Wait()
}

func getResolvableSelfHostname() (string, bool) {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "", false
	}

	// On some environments, hostnames obtained via os.Hostname are not resolvable.
	// For example, on my Macbook:
	// > $ host `hostname`
	// > Host sskvor-mac not found: 3(NXDOMAIN)
	// Let's try to resolve it.
	ips, err := net.LookupIP(hostname)
	if err != nil || len(ips) < 1 {
		return "", false
	}

	return hostname, true
}

////////////////////////////////////////////////////////////////////////////////

type PProfProfileSink struct {
	log         log.Logger
	address     string
	wantBrowser bool
}

func (s *PProfProfileSink) Store(profile []byte) error {
	mux := http.NewServeMux()

	server := func(args *driver.HTTPServerArgs) error {
		for k, v := range args.Handlers {
			mux.Handle(k, v)
		}
		return nil
	}

	options := &driver.Options{
		HTTPServer: server,
		Fetch:      &pprofProfileFetcher{profile},
		UI:         &dummyUI{s.log, s.wantBrowser},
		Flagset:    baseFlags(),
	}

	err := driver.PProf(options)
	if err != nil {
		return err
	}

	s.log.Info("Starting pprof http server", log.String("address", s.address))
	return http.ListenAndServe(s.address, mux)
}

type pprofProfileFetcher struct {
	profile []byte
}

func (p *pprofProfileFetcher) Fetch(src string, duration, timeout time.Duration) (*profile.Profile, string, error) {
	res, err := profile.Parse(bytes.NewBuffer(p.profile))
	return res, src, err
}

////////////////////////////////////////////////////////////////////////////////

type flags struct {
	bools       map[string]bool
	ints        map[string]int
	floats      map[string]float64
	strings     map[string]string
	args        []string
	stringLists map[string][]string
}

func (flags) ExtraUsage() string { return "" }

func (flags) AddExtraUsage(eu string) {}

func (f flags) Bool(s string, d bool, c string) *bool {
	if b, ok := f.bools[s]; ok {
		return &b
	}
	return &d
}

func (f flags) Int(s string, d int, c string) *int {
	if i, ok := f.ints[s]; ok {
		return &i
	}
	return &d
}

func (f flags) Float64(s string, d float64, c string) *float64 {
	if g, ok := f.floats[s]; ok {
		return &g
	}
	return &d
}

func (f flags) String(s, d, c string) *string {
	if t, ok := f.strings[s]; ok {
		return &t
	}
	return &d
}

func (f flags) StringList(s, d, c string) *[]*string {
	if t, ok := f.stringLists[s]; ok {
		// convert slice of strings to slice of string pointers before returning.
		tp := make([]*string, len(t))
		for i, v := range t {
			tp[i] = &v
		}
		return &tp
	}
	return &[]*string{}
}

func (f flags) Parse(func()) []string {
	return f.args
}

func baseFlags() flags {
	return flags{
		bools:  map[string]bool{},
		ints:   map[string]int{},
		floats: map[string]float64{},
		strings: map[string]string{
			"http":       "localhost:0",
			"symbolize":  "None",
			"no_browser": "",
		},
		args: []string{"http://example.com/pprof", "-no_browser"}, // dummy url to trigger fetching
	}
}

////////////////////////////////////////////////////////////////////////////////

type dummyUI struct {
	log         log.Logger
	wantBrowser bool
}

// IsTerminal implements driver.UI
func (u *dummyUI) IsTerminal() bool {
	return false
}

// Print implements driver.UI
func (u *dummyUI) Print(args ...interface{}) {
	u.log.Info("PProf UI message", log.Array("fields", args))
}

// PrintErr implements driver.UI
func (u *dummyUI) PrintErr(args ...interface{}) {
	u.log.Error("PProf UI error", log.Array("fields", args))
}

// ReadLine implements driver.UI
func (*dummyUI) ReadLine(prompt string) (string, error) {
	return "", nil
}

// SetAutoComplete implements driver.UI
func (*dummyUI) SetAutoComplete(func(string) string) {
}

// WantBrowser implements driver.UI
func (u *dummyUI) WantBrowser() bool {
	return u.wantBrowser
}

var _ driver.UI = (*dummyUI)(nil)
