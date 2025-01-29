// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package demangle

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

const rustFilename = "testdata/rust-demangle-expected"

// TestRustExpected is like TestExpected, but for Rust demangling.
// We ignore all test inputs that do not start with _R.
func TestRustExpected(t *testing.T) {
	t.Parallel()
	f, err := os.Open(rustFilename)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(f)
	lineno := 1
	for {
		format, got := getOptLine(t, scanner, &lineno)
		if !got {
			break
		}
		report := lineno
		input := getLine(t, scanner, &lineno)
		expect := getLine(t, scanner, &lineno)

		skip := false
		if len(format) > 0 && format[0] == '-' {
			for _, arg := range strings.Fields(format) {
				switch arg {
				case "--format=gnu-v3":
					skip = true
				case "--format=auto":
				case "--format=rust":
				default:
					t.Errorf("%s:%d: unrecognized argument %s", rustFilename, report, arg)
				}
			}
		}

		if skip {
			continue
		}

		oneRustTest(t, report, input, expect)
	}
	if err := scanner.Err(); err != nil {
		t.Error(err)
	}
}

// oneRustTest tests one entry from rust-demangle-expected.
func oneRustTest(t *testing.T, report int, input, expect string) {
	if *verbose {
		fmt.Println(input)
	}

	s, err := ToString(input)
	if err != nil {
		if err != ErrNotMangledName {
			if input == expect {
				return
			}
			t.Errorf("%s:%d: %v", rustFilename, report, err)
			return
		}
		s = input
	}

	if s != expect {
		t.Errorf("%s:%d: got %q, want %q", rustFilename, report, s, expect)
	}

	if s == expect && s != input && len(expect) > 64 {
		ss, err := ToString(input, MaxLength(6))
		if err != nil {
			t.Errorf("%s:%d: error with MaxLength: %v", rustFilename, report, err)
		} else if ss != expect[:64] {
			t.Errorf("%s:%d: MaxLength mismatch: %q != %q", rustFilename, report, ss, expect[:64])
		}
	}
}

const rustCheckFilename = "testdata/rust.test"

func TestRustCheck(t *testing.T) {
	t.Parallel()
	f, err := os.Open(rustCheckFilename)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(f)
	lineno := 1
	for scanner.Scan() {
		report := lineno
		line := strings.TrimSpace(scanner.Text())
		lineno++
		if !strings.HasPrefix(line, "CHECK: ") {
			continue
		}
		want := strings.TrimPrefix(line, "CHECK: ")
		if !scanner.Scan() {
			t.Fatalf("%s:%d: unexpected EOF", rustCheckFilename, report)
		}
		lineno++
		input := strings.TrimSpace(scanner.Text())

		got, err := ToString(input, LLVMStyle)
		if err != nil {
			if want != input {
				t.Errorf("%s:%d: %v", rustCheckFilename, report, err)
			}
		} else if got != want {
			t.Errorf("%s:%d: got %q, want %q", rustCheckFilename, report, got, want)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Error(err)
	}
}

const rustMangledTemplates = "_RNCINvMNtNtNtNtCs5myfTy8mnaF_6timely8dataflow9operators7generic10builder_rcINtB5_15OperatorBuilderINtNtNtBb_6scopes5child5ChildIB1z_INtNtBd_6worker6WorkerNtNtNtCsbo5udLplCaV_20timely_communication9allocator7generic7GenericENtNtCslnPiKci8RgF_7mz_repr9timestamp9TimestampEB3z_EE16build_rescheduleNCINvB4_5buildNCINvXNtB7_8operatorINtNtBb_6stream10StreamCoreB1y_INtNtCsfohDMHpnFpV_5alloc3vec3VecTTNtNtB3D_3row3RowB6k_EB3z_xEEEINtB52_8OperatorB1y_B5L_E14unary_frontierIB5M_INtNtB5Q_2rc2RcINtNtNtNtCsaEm0OTy3LfN_21differential_dataflow5trace15implementations3ord11OrdValBatchB6k_B6k_B3z_xjINtNtCsicJTUUNBAMQ_16timely_container11columnation11TimelyStackB6k_EB9o_EEENCINvXs1_NtNtNtB7V_9operators7arrange11arrangementINtNtB7V_10collection10CollectionB1y_B6j_xEINtBaK_7ArrangeB1y_B6k_B6k_xE12arrange_coreINtNtNtBb_8channels4pact12ExchangeCoreB5L_B6i_NCINvBaG_13arrange_namedINtNtB7R_12spine_fueled5SpineB7x_EE0EBdV_E0NCNCBaD_00BcN_E0NCNCB4Y_00E0NCNCB4K_00E0Cse28fqe15ASj_8clusterd"

func TestRustNoTemplaraParams(t *testing.T) {
	got, err := ToString(rustMangledTemplates, NoTemplateParams)
	if err != nil {
		t.Fatalf("ToString(%q) failed: %v", rustMangledTemplates, err)
	}
	want := "<timely::dataflow::operators::generic::builder_rc::OperatorBuilder<>>::build_reschedule::<>::{closure#0}"
	if got != want {
		t.Errorf("ToString(%q) = %q, want %q", rustMangledTemplates, got, want)
	}
}
