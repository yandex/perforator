package jvmscanner

import (
	"testing"
)

func TestU5decode(t *testing.T) {
	tests := []struct {
		name      string
		buf       []byte
		wantValue uint32
		wantN     int
		wantErr   bool
	}{
		{
			name:      "value 0",
			buf:       []byte{1},
			wantValue: 0,
			wantN:     1,
		},
		{
			name:      "value 1",
			buf:       []byte{2},
			wantValue: 1,
			wantN:     1,
		},
		{
			name:      "arbitrary mid single-byte value",
			buf:       []byte{100},
			wantValue: 99,
			wantN:     1,
		},
		{
			name:      "max single-byte value 190 (byte 191)",
			buf:       []byte{191},
			wantValue: 190,
			wantN:     1,
		},

		{
			name:      "min two-byte value 191",
			buf:       []byte{192, 1},
			wantValue: 191, // (192-1)*1 + (1-1)*64 = 191
			wantN:     2,
		},
		{
			name:      "two-byte value 255",
			buf:       []byte{192, 2},
			wantValue: 255, // 191 + 1*64 = 255
			wantN:     2,
		},
		{
			name:      "two-byte value 12351",
			buf:       []byte{192, 191},
			wantValue: 12351, // 191 + 190*64 = 191 + 12160 = 12351
			wantN:     2,
		},
		{
			name:      "max two-byte value 12414",
			buf:       []byte{255, 191},
			wantValue: 12414, // 254 + 190*64 = 254 + 12160 = 12414
			wantN:     2,
		},

		{
			name:      "min three-byte value 12415",
			buf:       []byte{192, 192, 1},
			wantValue: 12415, // 191 + 191*64 + 0 = 191 + 12224 = 12415
			wantN:     3,
		},
		{
			name:      "max three-byte value 794750",
			buf:       []byte{255, 255, 191},
			wantValue: 794750, // 254 + 254*64 + 190*4096 = 254 + 16256 + 778240
			wantN:     3,
		},

		{
			name:      "min four-byte value 794751",
			buf:       []byte{192, 192, 192, 1},
			wantValue: 794751, // 191 + 191*64 + 191*4096 + 0 = 191 + 12224 + 782336
			wantN:     4,
		},
		{
			name:      "max four-byte value 50864254",
			buf:       []byte{255, 255, 255, 191},
			wantValue: 50864254, // 254 + 254*64 + 254*4096 + 190*262144
			wantN:     4,
		},

		{
			name:      "min five-byte value 50864255",
			buf:       []byte{192, 192, 192, 192, 1},
			wantValue: 50864255, // 191*(1+64+4096+262144) + 0
			wantN:     5,
		},
		{
			name:      "five-byte with terminal last byte",
			buf:       []byte{255, 255, 255, 255, 191},
			wantValue: 3255312510, // 254*(1+64+4096+262144) + 190*16777216
			wantN:     5,
		},
		{
			name:      "five-byte forced stop on continuation last byte",
			buf:       []byte{192, 192, 192, 192, 192},
			wantValue: 3255312511, // 191*(1+64+4096+262144+16777216)
			wantN:     5,
		},
		{
			name:      "uint32 max value 4294967295",
			buf:       []byte{192, 254, 253, 253, 253},
			wantValue: 4294967295, // 191 + 253*64 + 252*4096 + 252*262144 + 252*16777216
			wantN:     5,
		},

		{
			name:      "trailing bytes after single-byte sequence",
			buf:       []byte{5, 10, 20},
			wantValue: 4,
			wantN:     1,
		},
		{
			name:      "trailing bytes after two-byte sequence",
			buf:       []byte{192, 2, 99},
			wantValue: 255,
			wantN:     2,
		},

		{
			name:    "zero byte at start",
			buf:     []byte{0},
			wantErr: true,
		},
		{
			name:    "zero byte after one continuation byte",
			buf:     []byte{192, 0},
			wantErr: true,
		},
		{
			name:    "zero byte after several continuation bytes",
			buf:     []byte{192, 192, 0},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, n, err := u5decode(tc.buf)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got val=%d n=%d", val, n)
					return
				}
				if err != errNullByte {
					t.Errorf("unexpected error: %v, want %v", err, errNullByte)
					return
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if val != tc.wantValue {
				t.Errorf("value: got %d, want %d", val, tc.wantValue)
			}
			if n != tc.wantN {
				t.Errorf("byte count: got %d, want %d", n, tc.wantN)
			}
		})
	}
}
