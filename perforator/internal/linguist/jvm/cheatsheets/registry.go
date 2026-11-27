package cheatsheets

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"

	"github.com/yandex/perforator/library/go/core/resource"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/jvm"
)

type Registry struct {
	byMajorVersion map[uint32][]*jvm.Cheatsheet
}

func compareVersions(lhs, rhs *jvm.Version) int {
	return cmp.Or(
		cmp.Compare(lhs.GetMajor(), rhs.GetMajor()),
		cmp.Compare(lhs.GetMinor(), rhs.GetMinor()),
		cmp.Compare(lhs.GetSecurity(), rhs.GetSecurity()),
	)
}

func newRegistryImpl(cheatsheets []*jvm.Cheatsheet) *Registry {
	bmv := make(map[uint32][]*jvm.Cheatsheet)
	for _, cs := range cheatsheets {
		version := cs.GetVersionFilter().GetMinVersion().GetMajor()
		bmv[version] = append(bmv[version], cs)
	}

	for _, cs := range bmv {
		slices.SortFunc(cs, func(lhs, rhs *jvm.Cheatsheet) int {
			return compareVersions(
				lhs.GetVersionFilter().GetMinVersion(),
				rhs.GetVersionFilter().GetMinVersion(),
			)
		})
	}

	return &Registry{
		byMajorVersion: bmv,
	}
}

func NewRegistry() *Registry {
	var cheatsheets []*jvm.Cheatsheet
	keys := resource.Keys()
	for _, k := range keys {
		if !strings.HasPrefix(k, "jvm-cheatsheets/") {
			continue
		}
		data := resource.MustGet(k)
		cs := new(jvm.Cheatsheet)
		err := prototext.Unmarshal(data, cs)
		if err != nil {
			panic(fmt.Sprintf("failed to parse cheatsheet %q: %v", k, err))
		}
		err = ValidateStatic(cs)
		if err != nil {
			panic(fmt.Sprintf("cheatsheet %q is invalid: %v", k, err))
		}
		cheatsheets = append(cheatsheets, cs)
	}
	return newRegistryImpl(cheatsheets)
}

func (r *Registry) Find(version *jvm.Version) *jvm.Cheatsheet {
	css := r.byMajorVersion[version.GetMajor()]
	pos, eq := slices.BinarySearchFunc(css, version, func(cs *jvm.Cheatsheet, ver *jvm.Version) int {
		csVer := cs.GetVersionFilter().GetMinVersion()
		return compareVersions(csVer, ver)
	})
	if !eq {
		if pos == 0 {
			return nil
		}
		pos -= 1
	}
	return css[pos]
}
