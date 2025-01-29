package registryutil

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/OneOfOne/xxhash"
)

var keyBufferPool = sync.Pool{New: func() any {
	return bytes.NewBuffer(make([]byte, 0, 512))
}}

// BuildRegistryKey creates registry name based on given prefix and tags
func BuildRegistryKey(prefix string, tags map[string]string) string {
	buffer := keyBufferPool.Get().(*bytes.Buffer)
	defer func() {
		buffer.Reset()
		keyBufferPool.Put(buffer)
	}()

	buffer.Write(strconv.AppendQuote(buffer.Bytes(), prefix))
	buffer.WriteString("{")
	StringifyTags(tags, buffer)
	buffer.WriteString("}")
	return buffer.String()
}

// BuildFQName returns name parts joined by given separator.
// Mainly used to append prefix to registry
func BuildFQName(separator string, parts ...string) (name string) {
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(separator)
		}
		b.WriteString(strings.Trim(p, separator))
	}
	return b.String()
}

// MergeTags merges 2 sets of tags with the tags from tagsRight overriding values from tagsLeft
func MergeTags(leftTags map[string]string, rightTags map[string]string) map[string]string {
	if leftTags == nil && rightTags == nil {
		return nil
	}

	if len(leftTags) == 0 {
		return rightTags
	}

	if len(rightTags) == 0 {
		return leftTags
	}

	newTags := make(map[string]string)
	for key, value := range leftTags {
		newTags[key] = value
	}
	for key, value := range rightTags {
		newTags[key] = value
	}
	return newTags
}

// StringifyTags returns string representation of given tags map.
// It is guaranteed that equal sets of tags will produce equal strings.
func StringifyTags(tags map[string]string, buffer *bytes.Buffer) {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for i, key := range keys {
		if i > 0 {
			buffer.WriteByte(',')
		}
		buffer.WriteString(key)
		buffer.WriteString("=")
		buffer.WriteString(tags[key])
	}
}

// VectorHash computes hash of metrics vector element
func VectorHash(tags map[string]string, labels []string) (uint64, error) {
	if len(tags) != len(labels) {
		return 0, errors.New("inconsistent tags and labels sets")
	}

	h := xxhash.New64()

	for _, label := range labels {
		v, ok := tags[label]
		if !ok {
			return 0, fmt.Errorf("label '%s' not found in tags", label)
		}
		_, _ = h.WriteString(label)
		_, _ = h.WriteString(v)
		_, _ = h.WriteString(",")
	}

	return h.Sum64(), nil
}
