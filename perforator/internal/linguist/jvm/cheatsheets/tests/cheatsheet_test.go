package cheatsheet_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/yandex/perforator/library/go/core/resource"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/jvm"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/cheatsheets"
	_ "github.com/yandex/perforator/perforator/internal/linguist/jvm/cheatsheets"
)

var supportedVersions = []string{
	"17",
	"21",
	"23",
	"24",
	"25",
}

func TestKeys(t *testing.T) {
	keys := resource.Keys()
	var actualVersions []string
	for _, key := range keys {
		t.Log(key)
		ver, ok := strings.CutPrefix(key, "jvm-cheatsheets/jdk")
		if !ok {
			continue
		}
		version := strings.TrimSuffix(ver, ".txtpb")
		actualVersions = append(actualVersions, version)
	}
	assert.ElementsMatch(t, supportedVersions, actualVersions)
}

func TestCheatsheetsAreValid(t *testing.T) {
	for _, ver := range supportedVersions {
		t.Run(ver, func(t *testing.T) {
			cheatsheet := new(jvm.Cheatsheet)
			data := resource.Get("jvm-cheatsheets/jdk" + ver + ".txtpb")
			require.NotNil(t, data)
			err := prototext.Unmarshal(data, cheatsheet)
			require.NoError(t, err)

			cheatsheet.AbstractInterpreterCode = new(uint64(42))
			cheatsheet.CodeCacheHeaps = new(uint64(42))

			// checks from jvmregistry.Registry.installBinaryConfig
			assert.NotNil(t, cheatsheet.FrameInterpreterFrameMethodOffset)
			assert.NotNil(t, cheatsheet.FrameReturnAddrOffset)

			intVer, err := strconv.Atoi(ver)
			require.NoError(t, err)

			err = cheatsheets.Validate(cheatsheet, uint32(intVer))
			assert.NoError(t, err)
		})
	}
}
