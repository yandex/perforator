package main

import (
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/btf"
	"golang.org/x/exp/maps"

	"github.com/yandex/perforator/library/go/slices"
)

func isType(entity btf.Type) bool {
	switch entity.(type) {
	case *btf.Struct, *btf.Enum, *btf.Union, *btf.Array, *btf.Pointer, *btf.Typedef, *btf.Int, *btf.Void:
		return true
	case *btf.Var, *btf.Func, *btf.FuncProto, *btf.Datasec, *btf.Fwd:
		return false
	}
	panic(fmt.Sprintf("unsupported case: %+T", entity))
}

func checkType(s2 *btf.Spec, typ1 btf.Type) error {
	name := typ1.TypeName()
	if name == "" {
		return nil
	}

	typ2s, err := s2.AnyTypesByName(name)
	if err != nil {
		return fmt.Errorf("failed to lookup entity %q in second spec: %w", name, err)
	}
	typ2s = slices.Filter(typ2s, isType)
	if len(typ2s) == 0 {
		return fmt.Errorf("type %q missing in second spec", name)
	}
	if len(typ2s) > 1 {
		return fmt.Errorf("found multiple types named %q in second spec", name)
	}
	typ2 := typ2s[0]
	sz1, err := btf.Sizeof(typ1)
	if err != nil {
		return fmt.Errorf("failed to get size of type %q in first spec: %w", name, err)
	}
	sz2, err := btf.Sizeof(typ2)
	if err != nil {
		return fmt.Errorf("failed to get size of type %q in second spec: %w", name, err)
	}
	if sz1 != sz2 {
		return fmt.Errorf("size of type %q differs in first and second spec: %d vs %d", name, sz1, sz2)
	}

	return nil
}

type consistencyChecker struct {
	actualInconsistencies   map[string]struct{}
	expectedInconsistencies map[string]struct{}
}

func (cc *consistencyChecker) checkBTFConsistency(s1, s2 *ebpf.CollectionSpec) error {
	it := s1.Types.Iterate()

	for it.Next() {
		typ1 := it.Type
		if !isType(typ1) {
			continue
		}
		err := checkType(s2.Types, typ1)
		if err != nil {
			_, ok := cc.expectedInconsistencies[typ1.TypeName()]
			if !ok {
				return err
			}
			cc.actualInconsistencies[typ1.TypeName()] = struct{}{}
		}
	}

	return nil
}

func (cc *consistencyChecker) finalize() error {
	if len(cc.actualInconsistencies) < len(cc.expectedInconsistencies) {
		return fmt.Errorf(
			"some known inconsistencies were fixed, time to update list: %v vs %v",
			maps.Keys(cc.expectedInconsistencies),
			maps.Keys(cc.actualInconsistencies),
		)
	}
	return nil
}
