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
	javaVersionLTE *uint32
}

func (r fieldReqs) describe() []string {
	var res []string
	if r.javaVersionGTE != nil {
		res = append(res, fmt.Sprintf("version >= %d", *r.javaVersionGTE))
	}
	if r.javaVersionLTE != nil {
		res = append(res, fmt.Sprintf("version <= %d", *r.javaVersionLTE))
	}
	return res
}

func makeReqs(f protoreflect.FieldDescriptor) (fieldReqs, error) {
	var reqs fieldReqs
	opts := f.Options().(*descriptorpb.FieldOptions)
	if opts == nil {
		return reqs, nil
	}
	gte := proto.GetExtension(opts, jvmproto.E_RequiredIfJavaGte).(uint32)
	if gte != 0 {
		reqs.javaVersionGTE = ptr.T(gte)
	}
	lte := proto.GetExtension(opts, jvmproto.E_RequiredIfJavaLte).(uint32)
	if lte != 0 {
		reqs.javaVersionLTE = ptr.T(lte)
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
		expected := true
		if reqs.javaVersionGTE != nil {
			expected = expected && version >= *reqs.javaVersionGTE
		}
		if reqs.javaVersionLTE != nil {
			expected = expected && version <= *reqs.javaVersionLTE
		}

		present := refl.Has(field)

		if expected && !present {
			return fmt.Errorf("invalid cheatsheet: field %q (%d) is required for version %d (reqs: %v)", field.Name(), field.Number(), version, reqs.describe())
		}
		if !expected && present {
			return fmt.Errorf("invalid cheatsheet: field %q (%d) is unsupported for version %d (reqs: %v)", field.Name(), field.Number(), version, reqs.describe())
		}
	}
	return nil
}
