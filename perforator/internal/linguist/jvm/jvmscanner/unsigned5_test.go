package jvmscanner

import (
	"testing"
)

func TestU5decode(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config
		buf       []byte
		wantValue uint32
		wantN     int
		wantErr   bool
	}{
		// u5config20p: zero byte is excluded (shift by 1).
		{
			name:      "20p value 0",
			cfg:       u5config20p,
			buf:       []byte{1},
			wantValue: 0,
			wantN:     1,
		},
		{
			name:      "20p value 1",
			cfg:       u5config20p,
			buf:       []byte{2},
			wantValue: 1,
			wantN:     1,
		},
		{
			name:      "20p arbitrary mid single-byte value",
			cfg:       u5config20p,
			buf:       []byte{100},
			wantValue: 99,
			wantN:     1,
		},
		{
			name:      "20p max single-byte value 190 (byte 191)",
			cfg:       u5config20p,
			buf:       []byte{191},
			wantValue: 190,
			wantN:     1,
		},

		{
			name:      "20p min two-byte value 191",
			cfg:       u5config20p,
			buf:       []byte{192, 1},
			wantValue: 191, // (192-1)*1 + (1-1)*64 = 191
			wantN:     2,
		},
		{
			name:      "20p two-byte value 255",
			cfg:       u5config20p,
			buf:       []byte{192, 2},
			wantValue: 255, // 191 + 1*64 = 255
			wantN:     2,
		},
		{
			name:      "20p two-byte value 12351",
			cfg:       u5config20p,
			buf:       []byte{192, 191},
			wantValue: 12351, // 191 + 190*64 = 191 + 12160 = 12351
			wantN:     2,
		},
		{
			name:      "20p max two-byte value 12414",
			cfg:       u5config20p,
			buf:       []byte{255, 191},
			wantValue: 12414, // 254 + 190*64 = 254 + 12160 = 12414
			wantN:     2,
		},

		{
			name:      "20p min three-byte value 12415",
			cfg:       u5config20p,
			buf:       []byte{192, 192, 1},
			wantValue: 12415, // 191 + 191*64 + 0 = 191 + 12224 = 12415
			wantN:     3,
		},
		{
			name:      "20p max three-byte value 794750",
			cfg:       u5config20p,
			buf:       []byte{255, 255, 191},
			wantValue: 794750, // 254 + 254*64 + 190*4096 = 254 + 16256 + 778240
			wantN:     3,
		},

		{
			name:      "20p min four-byte value 794751",
			cfg:       u5config20p,
			buf:       []byte{192, 192, 192, 1},
			wantValue: 794751, // 191 + 191*64 + 191*4096 + 0 = 191 + 12224 + 782336
			wantN:     4,
		},
		{
			name:      "20p max four-byte value 50864254",
			cfg:       u5config20p,
			buf:       []byte{255, 255, 255, 191},
			wantValue: 50864254, // 254 + 254*64 + 254*4096 + 190*262144
			wantN:     4,
		},

		{
			name:      "20p min five-byte value 50864255",
			cfg:       u5config20p,
			buf:       []byte{192, 192, 192, 192, 1},
			wantValue: 50864255, // 191*(1+64+4096+262144) + 0
			wantN:     5,
		},
		{
			name:      "20p five-byte with terminal last byte",
			cfg:       u5config20p,
			buf:       []byte{255, 255, 255, 255, 191},
			wantValue: 3255312510, // 254*(1+64+4096+262144) + 190*16777216
			wantN:     5,
		},
		{
			name:      "20p five-byte forced stop on continuation last byte",
			cfg:       u5config20p,
			buf:       []byte{192, 192, 192, 192, 192},
			wantValue: 3255312511, // 191*(1+64+4096+262144+16777216)
			wantN:     5,
		},
		{
			name:      "20p uint32 max value 4294967295",
			cfg:       u5config20p,
			buf:       []byte{192, 254, 253, 253, 253},
			wantValue: 4294967295, // 191 + 253*64 + 252*4096 + 252*262144 + 252*16777216
			wantN:     5,
		},

		{
			name:      "20p trailing bytes after single-byte sequence",
			cfg:       u5config20p,
			buf:       []byte{5, 10, 20},
			wantValue: 4,
			wantN:     1,
		},
		{
			name:      "20p trailing bytes after two-byte sequence",
			cfg:       u5config20p,
			buf:       []byte{192, 2, 99},
			wantValue: 255,
			wantN:     2,
		},

		{
			name:    "20p zero byte at start",
			cfg:     u5config20p,
			buf:     []byte{0},
			wantErr: true,
		},
		{
			name:    "20p zero byte after one continuation byte",
			cfg:     u5config20p,
			buf:     []byte{192, 0},
			wantErr: true,
		},
		{
			name:    "20p zero byte after several continuation bytes",
			cfg:     u5config20p,
			buf:     []byte{192, 192, 0},
			wantErr: true,
		},

		// u5config19m: no byte is excluded (shift by 0), zero byte is valid.
		{
			name:      "19m value 0 (byte 0)",
			cfg:       u5config19m,
			buf:       []byte{0},
			wantValue: 0,
			wantN:     1,
		},
		{
			name:      "19m value 1 (byte 1)",
			cfg:       u5config19m,
			buf:       []byte{1},
			wantValue: 1,
			wantN:     1,
		},
		{
			name:      "19m arbitrary mid single-byte value",
			cfg:       u5config19m,
			buf:       []byte{100},
			wantValue: 100,
			wantN:     1,
		},
		{
			name:      "19m max single-byte value 191 (byte 191)",
			cfg:       u5config19m,
			buf:       []byte{191},
			wantValue: 191,
			wantN:     1,
		},
		{
			name:      "19m min two-byte value 192",
			cfg:       u5config19m,
			buf:       []byte{192, 0},
			wantValue: 192, // 192*1 + 0*64 = 192
			wantN:     2,
		},
		{
			name:      "19m two-byte value with zero continuation byte",
			cfg:       u5config19m,
			buf:       []byte{192, 0},
			wantValue: 192,
			wantN:     2,
		},
		{
			name:      "19m two-byte value 256",
			cfg:       u5config19m,
			buf:       []byte{192, 1},
			wantValue: 256, // 192 + 1*64 = 256
			wantN:     2,
		},
		{
			name:      "19m three-byte value with zero continuation byte",
			cfg:       u5config19m,
			buf:       []byte{192, 192, 0},
			wantValue: 12480, // 192 + 192*64 + 0*4096 = 192 + 12288 = 12480
			wantN:     3,
		},
		{
			name:      "19m trailing bytes after single-byte sequence",
			cfg:       u5config19m,
			buf:       []byte{5, 10, 20},
			wantValue: 5,
			wantN:     1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, n, err := u5decode(tc.cfg, tc.buf)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got val=%d n=%d", val, n)
					return
				}
				if err != errExcludedByte {
					t.Errorf("unexpected error: %v, want %v", err, errExcludedByte)
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
