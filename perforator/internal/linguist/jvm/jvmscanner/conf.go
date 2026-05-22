package jvmscanner

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/yandex/perforator/library/go/ptr"
	jvmproto "github.com/yandex/perforator/perforator/agent/preprocessing/proto/jvm"
)

type fieldReqs struct {
	javaVersionGTE *uint32
}

func makeReqs(f protoreflect.FieldDescriptor) (fieldReqs, error) {
	var reqs fieldReqs
	opts := f.Options().(*descriptorpb.FieldOptions)
	if opts == nil {
		return reqs, nil
	}
	ext := proto.GetExtension(opts, jvmproto.E_RequiredIfJavaGte)
	if ext != nil {
		reqs.javaVersionGTE = ptr.T(ext.(uint32))
	}

	return reqs, nil
}

func validateProto(cs *jvmproto.Cheatsheet, version uint32) error {
	refl := cs.ProtoReflect()
	fields := refl.Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)

		reqs, err := makeReqs(field)
		if err != nil {
			return fmt.Errorf("internal error: failed to process validation options for field %q (%d): %w", field.Name(), field.Number(), err)
		}
		var expected bool
		if reqs.javaVersionGTE != nil {
			expected = version >= *reqs.javaVersionGTE
		} else {
			expected = true
		}

		present := refl.Has(field)

		if expected && !present {
			return fmt.Errorf("invalid cheatsheet: field %q (%d) is required for version %d", field.Name(), field.Number(), version)
		}
		if !expected && present {
			return fmt.Errorf("invalid cheatsheet: field %q (%d) is unsupported for version %d", field.Name(), field.Number(), version)
		}
	}
	return nil
}
