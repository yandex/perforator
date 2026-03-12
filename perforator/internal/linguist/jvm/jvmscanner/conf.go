package jvmscanner

import (
	"fmt"

	jvmproto "github.com/yandex/perforator/perforator/agent/preprocessing/proto/jvm"
)

func validateProto(cs *jvmproto.Cheatsheet) error {
	refl := cs.ProtoReflect()
	fields := refl.Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		// Currently all fields in cheatsheet are required
		if !refl.Has(field) {
			return fmt.Errorf("invalid cheatsheet: required field %q (%d) is missing", field.Name(), field.Number())
		}
	}
	return nil
}
