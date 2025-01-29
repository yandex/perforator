package perfmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOk(t *testing.T) {
	conf, errs := parseProcessConfig("percentage=42,java=true")
	assert.Empty(t, errs)
	assert.Equal(t, uint32(42), conf.percentage)
	assert.True(t, conf.java)
}

func TestParseUnknownField(t *testing.T) {
	conf, errs := parseProcessConfig("percentage=42,unknown=true")
	assert.Equal(t, 1, len(errs))
	assert.Equal(t, uint32(42), conf.percentage)
}

func TestParseInvalidFormat(t *testing.T) {
	// should not panic
	parseProcessConfig(",,kek,,,,42=foo,percentage,java")
}
