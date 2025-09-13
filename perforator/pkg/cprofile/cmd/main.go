package main

import (
	"os"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/perforator/pkg/cprofile"
	"github.com/yandex/perforator/perforator/proto/profile"
)

func main() {
	mgr, err := cprofile.NewMergeManager(16)
	if err != nil {
		panic(err)
	}

	session, err := mgr.Start(&profile.MergeOptions{
		LabelFilter: &profile.LabelFilter{
			SkippedKeyPrefixes: []string{"tls", "cgroup"},
		},
	})
	if err != nil {
		panic(err)
	}

	var g errgroup.Group
	g.SetLimit(32)

	for _, arg := range os.Args[1:] {
		g.Go(func() error {
			data, err := os.ReadFile(arg)
			if err != nil {
				return err
			}
			return session.AddPProfProfile(data)
		})
	}

	err = g.Wait()
	if err != nil {
		panic(err)
	}

	profile, err := session.Finish()
	if err != nil {
		panic(err)
	}
	defer profile.Free()

	data, err := profile.MarshalPProf()
	if err != nil {
		panic(err)
	}

	_, err = os.Stdout.Write(data)
	if err != nil {
		panic(err)
	}
}
