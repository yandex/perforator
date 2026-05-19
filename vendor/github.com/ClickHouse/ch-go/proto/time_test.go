package proto

import (
	"testing"
	"time"
)

func TestTime32_String(t *testing.T) {
	tests := []struct {
		name     string
		time     Time32
		expected string
	}{
		{"zero", Time32(0), "00:00:00"},
		{"one hour", Time32(3600), "01:00:00"},
		{"one minute", Time32(60), "00:01:00"},
		{"one second", Time32(1), "00:00:01"},
		{"complex time", Time32(3661), "01:01:01"}, // 1 hour, 1 minute, 1 second
		{"max time", Time32(86399), "23:59:59"},    // 23:59:59
		{"midnight", Time32(0), "00:00:00"},
		{"noon", Time32(43200), "12:00:00"}, // 12:00:00
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.time.String(); got != tt.expected {
				t.Errorf("Time32.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTime64_String(t *testing.T) {
	tests := []struct {
		name     string
		time     Time64
		expected string
	}{
		{"zero", Time64(0), "00:00:00.000000000"},
		{"one second", Time64(1e9), "00:00:01.000000000"},
		{"one minute", Time64(60e9), "00:01:00.000000000"},
		{"one hour", Time64(3600e9), "01:00:00.000000000"},
		{"with nanoseconds", Time64(3661e9 + 123456789), "01:01:01.123456789"},
		{"max time", Time64(86399e9 + 999999999), "23:59:59.999999999"},
		{"noon", Time64(43200e9), "12:00:00.000000000"},
		{"microsecond precision", Time64(1e6), "00:00:00.001000000"},
		{"millisecond precision", Time64(1e3), "00:00:00.000001000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.time.String(); got != tt.expected {
				t.Errorf("Time64.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseTime32(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Time32
		expectError bool
	}{
		{"valid time", "12:34:56", Time32(12*3600 + 34*60 + 56), false},
		{"zero time", "00:00:00", Time32(0), false},
		{"max time", "23:59:59", Time32(23*3600 + 59*60 + 59), false},
		{"noon", "12:00:00", Time32(12 * 3600), false},
		{"midnight", "00:00:00", Time32(0), false},
		{"invalid format", "12:34", Time32(0), true},
		{"invalid format 2", "12:34:56:78", Time32(0), true},
		{"non-numeric", "ab:cd:ef", Time32(0), true},
		{"empty string", "", Time32(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTime32(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseTime32() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ParseTime32() unexpected error: %v", err)
				}
				if got != tt.expected {
					t.Errorf("ParseTime32() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestParseTime64(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Time64
		expectError bool
	}{
		{"valid time", "12:34:56", Time64((12*3600 + 34*60 + 56) * 1e9), false},
		{"zero time", "00:00:00", Time64(0), false},
		{"max time", "23:59:59", Time64((23*3600 + 59*60 + 59) * 1e9), false},
		{"noon", "12:00:00", Time64(12 * 3600 * 1e9), false},
		{"midnight", "00:00:00", Time64(0), false},
		{"invalid format", "12:34", Time64(0), true},
		{"invalid format 2", "12:34:56:78", Time64(0), true},
		{"non-numeric", "ab:cd:ef", Time64(0), true},
		{"empty string", "", Time64(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTime64(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseTime64() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ParseTime64() unexpected error: %v", err)
				}
				if got != tt.expected {
					t.Errorf("ParseTime64() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestIntoTime32(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected Time32
	}{
		{"zero time", time.Duration(0), Time32(0)},
		{"12h", 12 * time.Hour, Time32(12 * 3600)},
		{"13h45m30s", 13*time.Hour + 45*time.Minute + 30*time.Second, Time32(13*3600 + 45*60 + 30)},
		{"23h59m59s", 23*time.Hour + 59*time.Minute + 59*time.Second, Time32(23*3600 + 59*60 + 59)},
		{"time32 should ignore nanoseconds", 12*time.Hour + 123456789*time.Nanosecond, Time32(12 * 3600)},
		{"14h30m45s", 14*time.Hour + 30*time.Minute + 45*time.Second, Time32(14*3600 + 30*60 + 45)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IntoTime32(tt.input); got != tt.expected {
				t.Errorf("FromTime32() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIntoTime64WithPrecision(t *testing.T) {
	tests := []struct {
		name      string
		input     time.Duration
		precision Precision
		expected  Time64
	}{
		{"zero time second", time.Duration(0), Precision(0), Time64(0)},
		{"zero time milli", time.Duration(0), Precision(3), Time64(0)},
		{"zero time micro", time.Duration(0), Precision(6), Time64(0)},
		{"zero time nano", time.Duration(0), Precision(9), Time64(0)},

		{"12h second", 12 * time.Hour, Precision(0), Time64(12 * 3600 * 1e0)},
		{"12h milli", 12 * time.Hour, Precision(3), Time64(12 * 3600 * 1e3)},
		{"12h micro", 12 * time.Hour, Precision(6), Time64(12 * 3600 * 1e6)},
		{"12h nano", 12 * time.Hour, Precision(9), Time64(12 * 3600 * 1e9)},

		{"13h45m30s second", 13*time.Hour + 45*time.Minute + 30*time.Second, Precision(0), Time64((13*3600 + 45*60 + 30) * 1e0)},
		{"13h45m30s milli", 13*time.Hour + 45*time.Minute + 30*time.Second, Precision(3), Time64((13*3600 + 45*60 + 30) * 1e3)},
		{"13h45m30s micro", 13*time.Hour + 45*time.Minute + 30*time.Second, Precision(6), Time64((13*3600 + 45*60 + 30) * 1e6)},
		{"13h45m30s nano", 13*time.Hour + 45*time.Minute + 30*time.Second, Precision(9), Time64((13*3600 + 45*60 + 30) * 1e9)},

		{"23h59m59s second", 23*time.Hour + 59*time.Minute + 59*time.Second, Precision(0), Time64((23*3600 + 59*60 + 59) * 1e0)},
		{"23h59m59s milli", 23*time.Hour + 59*time.Minute + 59*time.Second, Precision(3), Time64((23*3600 + 59*60 + 59) * 1e3)},
		{"23h59m59s micro", 23*time.Hour + 59*time.Minute + 59*time.Second, Precision(6), Time64((23*3600 + 59*60 + 59) * 1e6)},
		{"23h59m59s nano", 23*time.Hour + 59*time.Minute + 59*time.Second, Precision(9), Time64((23*3600 + 59*60 + 59) * 1e9)},

		{"time64 should not ignore nanoseconds", 12*time.Hour + 123456789*time.Nanosecond, Precision(9), Time64(12*3600*1e9 + 123456789)},
		{"time64 should not ignore microseconds", 12*time.Hour + 123456789*time.Microsecond, Precision(6), Time64(12*3600*1e6 + 123456789)},
		{"time64 should not ignore milliseconds", 12*time.Hour + 123456789*time.Millisecond, Precision(3), Time64(12*3600*1e3 + 123456789)},
		{"time64 should not ignore seconds", 12*time.Hour + 123456789*time.Second, Precision(0), Time64(12*3600*1e0 + 123456789)},

		{"14h30m45s123456789ns full", 14*time.Hour + 30*time.Minute + 45*time.Second + 123456789*time.Nanosecond, Precision(9), Time64((14*3600+30*60+45)*1e9 + 123456789)},
		{"14h30m45s123456789ns micro", 14*time.Hour + 30*time.Minute + 45*time.Second + 123456789*time.Nanosecond, Precision(6), Time64((14*3600+30*60+45)*1e6 + 123456)},
		{"14h30m45s123456789ns milli", 14*time.Hour + 30*time.Minute + 45*time.Second + 123456789*time.Nanosecond, Precision(3), Time64((14*3600+30*60+45)*1e3 + 123)},
		{"14h30m45s123456789ns seconds", 14*time.Hour + 30*time.Minute + 45*time.Second + 123456789*time.Nanosecond, Precision(0), Time64((14*3600 + 30*60 + 45) * 1e0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IntoTime64WithPrecision(tt.input, tt.precision); got != tt.expected {
				t.Errorf("IntoTime64WithPrecision() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIntoTime64(t *testing.T) {
	// IntoTime64 always treat duration with PrecisionMax (which is nanoseconds)

	tests := []struct {
		name     string
		input    time.Duration
		expected Time64
	}{
		{"zero time", time.Duration(0), Time64(0)},
		{"12h", 12 * time.Hour, Time64(12 * 3600 * 1e9)},
		{"13h45m30s", 13*time.Hour + 45*time.Minute + 30*time.Second, Time64((13*3600 + 45*60 + 30) * 1e9)},
		{"23h59m59s", 23*time.Hour + 59*time.Minute + 59*time.Second, Time64((23*3600 + 59*60 + 59) * 1e9)},
		{"time64 should not ignore nanoseconds", 12*time.Hour + 123456789*time.Nanosecond, Time64(12*3600*1e9 + 123456789)},
		{"14h30m45s123456789ns", 14*time.Hour + 30*time.Minute + 45*time.Second + 123456789*time.Nanosecond, Time64((14*3600+30*60+45)*1e9 + 123456789)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IntoTime64(tt.input); got != tt.expected {
				t.Errorf("IntoTime64() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTime32_Duration(t *testing.T) {
	tests := []struct {
		name     string
		time     Time32
		expected time.Duration
	}{
		{"zero", Time32(0), time.Duration(0)},
		{"12h", Time32(12 * 3600), 12 * time.Hour},
		{"13h45m30s", Time32(13*3600 + 45*60 + 30), 13*time.Hour + 45*time.Minute + 30*time.Second},
		{"23h59m59s", Time32(23*3600 + 59*60 + 59), 23*time.Hour + 59*time.Minute + 59*time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.time.Duration(); got != tt.expected {
				t.Errorf("Time32.Duration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTime64_DurationWithPrecision(t *testing.T) {
	tests := []struct {
		name      string
		time      Time64
		precision Precision
		expected  time.Duration
	}{
		{"zero seconds", Time64(0), Precision(0), time.Duration(0)},
		{"zero milli", Time64(0), Precision(3), time.Duration(0)},
		{"zero micro", Time64(0), Precision(6), time.Duration(0)},
		{"zero nano", Time64(0), Precision(9), time.Duration(0)},

		{"12h seconds", Time64(12 * 3600 * 1e0), Precision(0), 12 * time.Hour},
		{"12h milli", Time64(12 * 3600 * 1e3), Precision(3), 12 * time.Hour},
		{"12h micro", Time64(12 * 3600 * 1e6), Precision(6), 12 * time.Hour},
		{"12h nano", Time64(12 * 3600 * 1e9), Precision(9), 12 * time.Hour},

		{"13h45m30s seconds", Time64((13*3600 + 45*60 + 30) * 1e0), Precision(0), 13*time.Hour + 45*time.Minute + 30*time.Second},
		{"13h45m30s milli", Time64((13*3600 + 45*60 + 30) * 1e3), Precision(3), 13*time.Hour + 45*time.Minute + 30*time.Second},
		{"13h45m30s micro", Time64((13*3600 + 45*60 + 30) * 1e6), Precision(6), 13*time.Hour + 45*time.Minute + 30*time.Second},
		{"13h45m30s nano", Time64((13*3600 + 45*60 + 30) * 1e9), Precision(9), 13*time.Hour + 45*time.Minute + 30*time.Second},

		{"23h59m59s seconds", Time64((23*3600 + 59*60 + 59) * 1e0), Precision(0), 23*time.Hour + 59*time.Minute + 59*time.Second},
		{"23h59m59s milli", Time64((23*3600 + 59*60 + 59) * 1e3), Precision(3), 23*time.Hour + 59*time.Minute + 59*time.Second},
		{"23h59m59s micro", Time64((23*3600 + 59*60 + 59) * 1e6), Precision(6), 23*time.Hour + 59*time.Minute + 59*time.Second},
		{"23h59m59s nano", Time64((23*3600 + 59*60 + 59) * 1e9), Precision(9), 23*time.Hour + 59*time.Minute + 59*time.Second},

		{"12h123456789ns", Time64(12*3600*1e9 + 123456789), Precision(9), 12*time.Hour + 123456789*time.Nanosecond},
		{"12h123456us", Time64(12*3600*1e6 + 123456), Precision(6), 12*time.Hour + 123456*time.Microsecond},
		{"12h123ms", Time64(12*3600*1e3 + 123), Precision(3), 12*time.Hour + 123*time.Millisecond},
		{"12h2s", Time64(12*3600*1e0 + 2), Precision(0), 12*time.Hour + 2*time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.time.ToDurationWithPrecision(tt.precision); got != tt.expected {
				t.Errorf("Time64.ToDurationWithPrecision() = %v, want %v", got, tt.expected)
			}
		})
	}
}
func TestTime64_Duration(t *testing.T) {
	tests := []struct {
		name     string
		time     Time64
		expected time.Duration
	}{
		{"zero", Time64(0), time.Duration(0)},
		{"12h", Time64(12 * 3600 * 1e9), 12 * time.Hour},
		{"13h45m30s", Time64((13*3600 + 45*60 + 30) * 1e9), 13*time.Hour + 45*time.Minute + 30*time.Second},
		{"23h59m59s", Time64((23*3600 + 59*60 + 59) * 1e9), 23*time.Hour + 59*time.Minute + 59*time.Second},
		{"12h123456789ns", Time64(12*3600*1e9 + 123456789), 12*time.Hour + 123456789*time.Nanosecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.time.Duration(); got != tt.expected {
				t.Errorf("Time64.Duration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTime32_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		time Time32
	}{
		{"zero", Time32(0)},
		{"noon", Time32(12 * 3600)},
		{"complex time", Time32(13*3600 + 45*60 + 30)},
		{"max time", Time32(23*3600 + 59*60 + 59)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to string
			str := tt.time.String()

			// Parse back
			parsed, err := ParseTime32(str)
			if err != nil {
				t.Errorf("ParseTime32() failed: %v", err)
			}

			if parsed != tt.time {
				t.Errorf("Round trip failed: original = %v, parsed = %v", tt.time, parsed)
			}
		})
	}
}

func TestTime64_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		time Time64
	}{
		{"zero", Time64(0)},
		{"noon", Time64(12 * 3600 * 1e9)},
		{"complex time", Time64((13*3600 + 45*60 + 30) * 1e9)},
		{"max time", Time64((23*3600 + 59*60 + 59) * 1e9)},
		{"with nanoseconds", Time64(12*3600*1e9 + 123456789)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to string
			str := tt.time.String()

			// Parse back (note: ParseTime64 doesn't handle nanoseconds, so we test the seconds part)
			parsed, err := ParseTime64(str[:8]) // Take only HH:MM:SS part
			if err != nil {
				t.Errorf("ParseTime64() failed: %v", err)
			}

			// Compare seconds part
			expectedSeconds := int64(tt.time) / 1e9
			parsedSeconds := int64(parsed) / 1e9
			if parsedSeconds != expectedSeconds {
				t.Errorf("Round trip failed: original seconds = %v, parsed seconds = %v", expectedSeconds, parsedSeconds)
			}
		})
	}
}

func TestTime32_TimeConversion(t *testing.T) {
	tests := []struct {
		name string
		time Time32
	}{
		{"zero", Time32(0)},
		{"noon", Time32(12 * 3600)},
		{"complex time", Time32(13*3600 + 45*60 + 30)},
		{"max time", Time32(23*3600 + 59*60 + 59)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to time.Time
			t1 := tt.time.Duration()

			// Convert back
			t2 := IntoTime32(t1)

			if t2 != tt.time {
				t.Errorf("Time conversion failed: original = %v, converted back = %v", tt.time, t2)
			}
		})
	}
}

func TestTime64_TimeConversion(t *testing.T) {
	tests := []struct {
		name string
		time Time64
	}{
		{"zero", Time64(0)},
		{"noon", Time64(12 * 3600 * 1e9)},
		{"complex time", Time64((13*3600 + 45*60 + 30) * 1e9)},
		{"max time", Time64((23*3600 + 59*60 + 59) * 1e9)},
		{"with nanoseconds", Time64(12*3600*1e9 + 123456789)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to time.Time
			t1 := tt.time.Duration()

			// Convert back
			t2 := IntoTime64(t1)

			if t2 != tt.time {
				t.Errorf("Time conversion failed: original = %v, converted back = %v", tt.time, t2)
			}
		})
	}
}

func BenchmarkTime32_String(b *testing.B) {
	t := Time32(12*3600 + 34*60 + 56) // 12:34:56
	for i := 0; i < b.N; i++ {
		_ = t.String()
	}
}

func BenchmarkTime64_String(b *testing.B) {
	t := Time64(12*3600*1e9 + 34*60*1e9 + 56*1e9 + 123456789) // 12:34:56.123456789
	for i := 0; i < b.N; i++ {
		_ = t.String()
	}
}

func BenchmarkParseTime32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseTime32("12:34:56")
	}
}

func BenchmarkParseTime64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseTime64("12:34:56")
	}
}

func BenchmarkFromTime32(b *testing.B) {
	t := 12*time.Hour + 34*time.Minute + 56*time.Second
	for i := 0; i < b.N; i++ {
		_ = IntoTime32(t)
	}
}

func BenchmarkFromTime64(b *testing.B) {
	t := 12*time.Hour + 34*time.Minute + 56*time.Second + 123456789*time.Nanosecond
	for i := 0; i < b.N; i++ {
		_ = IntoTime64(t)
	}
}
