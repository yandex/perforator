package pgo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testTool(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	path := filepath.Join(t.TempDir(), "tool")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0700))
	return path
}

func TestLLVMToolArguments(t *testing.T) {
	tool := testTool(t, `for arg do
 if [ "$next" = yes ]; then out=$arg; next=no; fi
 case "$arg" in --out|-o) next=yes;; esac
done
printf '%s\n' "$@" > "$out"
cat >> "$out"
`)
	for _, format := range []Format{FormatAutoFDO, FormatBOLT} {
		t.Run(string(format), func(t *testing.T) {
			data, err := NewTools(WithCreateLLVMProfPath(tool), WithLLVMBoltPath(tool)).Convert(context.Background(), format, "/tmp/app with spaces", []byte("branch-data"))
			require.NoError(t, err)
			require.Contains(t, string(data), "/tmp/app with spaces\n")
			require.Contains(t, string(data), "branch-data")
			if format == FormatBOLT {
				require.Contains(t, string(data), "--aggregate-only\n")
				require.Contains(t, string(data), "yaml\n")
			} else {
				require.Contains(t, string(data), "extbinary\n")
			}
		})
	}
}

func TestLLVMFailuresAndCancellation(t *testing.T) {
	tool := testTool(t, "echo invalid-ELF >&2\nexit 7\n")
	_, err := NewTools(WithLLVMBoltPath(tool)).Convert(context.Background(), FormatBOLT, "app", nil)
	require.EqualError(t, err, tool+" failed: exit status 7")
	_, err = NewTools(WithLLVMBoltPath("/no/such/llvm-bolt")).Convert(context.Background(), FormatBOLT, "app", nil)
	require.ErrorContains(t, err, "install")
	tool = testTool(t, "exit 0\n")
	_, err = NewTools(WithLLVMBoltPath(tool)).Convert(context.Background(), FormatBOLT, "app", nil)
	require.ErrorContains(t, err, "empty profile")
	tool = testTool(t, "exec sleep 30\n")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = NewTools(WithLLVMBoltPath(tool)).Convert(ctx, FormatBOLT, "app", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestLLVMCancellationCause(t *testing.T) {
	tool := testTool(t, "exec sleep 30\n")
	cause := errors.New("profile generation interrupted")
	ctx, cancel := context.WithTimeoutCause(context.Background(), 100*time.Millisecond, cause)
	defer cancel()
	_, err := NewTools(WithLLVMBoltPath(tool)).Convert(ctx, FormatBOLT, "app", nil)
	require.ErrorIs(t, err, cause)
}

func TestLLVMStderr(t *testing.T) {
	for _, exitCode := range []int{0, 7} {
		t.Run(fmt.Sprint(exitCode), func(t *testing.T) {
			tool := testTool(t, fmt.Sprintf(`for arg do
 if [ "$next" = yes ]; then out=$arg; next=no; fi
 case "$arg" in --out|-o) next=yes;; esac
done
printf 'profile' > "$out"
cat >&2
exit %d
`, exitCode))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			input := strings.Repeat("x", 128*1024) + "diagnostic tail\n"
			var stderr bytes.Buffer
			tools := NewTools(WithLLVMBoltPath(tool), WithStderr(&stderr))
			profile, err := tools.Convert(ctx, FormatBOLT, "app", []byte(input))
			if exitCode == 0 {
				require.NoError(t, err)
				require.Equal(t, "profile", string(profile))
			} else {
				require.EqualError(t, err, tool+" failed: exit status 7")
			}
			require.Equal(t, input, stderr.String())
		})
	}
}
