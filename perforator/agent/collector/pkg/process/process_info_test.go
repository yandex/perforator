package process

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/internal/unwinder"
)

func TestNewProcessInfo(t *testing.T) {
	pi := newProcessInfo()

	require.Equal(t, pi.MainBinaryId, unwinder.BinaryId(math.MaxUint64))
	require.Equal(t, pi.UnwindType, unwinder.UnwindTypeDwarf)

	v := reflect.ValueOf(pi).Elem()
	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := typ.Field(i)

		if field.Type() == reflect.TypeOf(unwinder.MappedBinary{}) {
			baseAddr := field.FieldByName("BaseAddress")
			if !baseAddr.IsValid() {
				t.Errorf("field %s does not have BaseAddress", fieldType.Name)
				continue
			}

			if baseAddr.Uint() != math.MaxUint64 {
				t.Errorf("field %s BaseAddress = %d, want %d", fieldType.Name, baseAddr.Uint(), uint64(math.MaxUint64))
			}
		}
	}
}
