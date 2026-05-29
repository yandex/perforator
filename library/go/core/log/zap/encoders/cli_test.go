package encoders

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var cliTestCfg = zapcore.EncoderConfig{
	MessageKey:     "M",
	LevelKey:       "L",
	TimeKey:        "T",
	NameKey:        "N",
	CallerKey:      "C",
	StacktraceKey:  "S",
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

func TestCLIEncodeEntry(t *testing.T) {
	tests := []struct {
		desc     string
		expected string
		ent      zapcore.Entry
		fields   []zapcore.Field
	}{
		{
			desc:     "message with per-call fields",
			expected: "lob law so=passes answer=42\n",
			ent:      zapcore.Entry{Message: "lob law"},
			fields: []zapcore.Field{
				zap.String("so", "passes"),
				zap.Int("answer", 42),
			},
		},
		{
			// CLI format omits time, level, caller, name — only the raw message is written.
			desc:     "entry metadata is omitted",
			expected: "msg\n",
			ent: zapcore.Entry{
				Level:      zapcore.InfoLevel,
				Time:       time.Date(2018, 6, 19, 16, 33, 42, 99, time.UTC),
				LoggerName: "bob",
				Message:    "msg",
				Caller:     zapcore.EntryCaller{Defined: true, File: "foo.go", Line: 42},
			},
		},
		{
			// Message is written via buf.AppendString (unquoted), not through the
			// encoder's AppendString, so single quotes pass through unchanged.
			desc:     "message is written unquoted",
			expected: "it's a 'test'\n",
			ent:      zapcore.Entry{Message: "it's a 'test'"},
		},
		{
			// The stack is appended via the array-encoder AppendString, which adds a
			// comma separator (last byte before the stack is '\n') and quotes values
			// containing spaces.
			desc:     "stacktrace appended after newline",
			expected: "oops\n,'panic: runtime error'\n",
			ent: zapcore.Entry{
				Message: "oops",
				Stack:   "panic: runtime error",
			},
		},
	}

	enc, _ := NewCliEncoder(cliTestCfg)
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			buf, err := enc.EncodeEntry(tt.ent, tt.fields)
			if assert.NoError(t, err, "Unexpected CLI encoding error.") {
				assert.Equal(t, tt.expected, buf.String(), "Incorrect encoded CLI entry.")
			}
			buf.Free()
		})
	}
}

// TestCLIEncodeEntry_PresetFields verifies that fields added to the encoder
// before EncodeEntry (i.e. via logger.With) appear between the message and the
// per-call fields, with single-space separators throughout.
func TestCLIEncodeEntry_PresetFields(t *testing.T) {
	enc := newCliEncoder(cliTestCfg)
	enc.AddString("component", "auth")

	buf, err := enc.EncodeEntry(
		zapcore.Entry{Message: "user logged in"},
		[]zapcore.Field{zap.String("user", "alice")},
	)
	if assert.NoError(t, err) {
		assert.Equal(t, "user logged in component=auth user=alice\n", buf.String())
	}
	buf.Free()
}

func TestCLIEncodeEntry_CustomLineEnding(t *testing.T) {
	cfg := cliTestCfg
	cfg.LineEnding = "\r\n"
	enc, _ := NewCliEncoder(cfg)

	buf, err := enc.EncodeEntry(zapcore.Entry{Message: "msg"}, nil)
	if assert.NoError(t, err) {
		assert.Equal(t, "msg\r\n", buf.String())
	}
	buf.Free()
}

// TestCLIEncoderClone verifies that clones carry a copy of the pre-set fields
// and that further additions to each clone do not bleed into sibling clones.
func TestCLIEncoderClone(t *testing.T) {
	enc := newCliEncoder(cliTestCfg)
	enc.AddString("base", "val")

	c1 := enc.Clone().(*cliEncoder)
	c2 := enc.Clone().(*cliEncoder)
	c1.AddString("only1", "x")
	c2.AddString("only2", "y")

	b1, _ := c1.EncodeEntry(zapcore.Entry{Message: "msg"}, nil)
	b2, _ := c2.EncodeEntry(zapcore.Entry{Message: "msg"}, nil)
	defer b1.Free()
	defer b2.Free()

	assert.Equal(t, "msg base=val only1=x\n", b1.String())
	assert.Equal(t, "msg base=val only2=y\n", b2.String())
}
