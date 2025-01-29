package registryutil

import (
	"bytes"
	"fmt"
	"testing"
)

func BenchmarkRegistryKey(b *testing.B) {
	data := map[string]string{
		"location": "location",
		"code":     "code",
		"status":   "status",
		"field":    "field",
		"field2":   "field2",
		"field3":   "field2",
		"field4":   "field2",
		"field45":  "field2",
	}

	largeData := make(map[string]string)
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("key-%d", i)] = string(bytes.Repeat([]byte("a"), 1024))
	}

	b.Run("actual", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			BuildRegistryKey("some-prefix", data)
		}
	})

	b.Run("actual-large-data", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			BuildRegistryKey("some-prefix", largeData)
		}
	})
}
