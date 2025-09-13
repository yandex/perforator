package internal_test

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"

	"github.com/jarcoal/httpmock/internal"
)

func TestErrorNoResponderFoundMistake(t *testing.T) {
	e := &internal.ErrorNoResponderFoundMistake{
		Kind:      "method",
		Orig:      "pipo",
		Suggested: "BINGO",
	}
	td.Cmp(t, e.Error(), `no responder found for method "pipo", but one matches method "BINGO"`)
	td.Cmp(t, e.Unwrap(), internal.NoResponderFound)

	e = &internal.ErrorNoResponderFoundMistake{
		Kind:      "matcher",
		Orig:      "--not--used--",
		Suggested: "BINGO",
	}
	td.Cmp(t, e.Error(), `no responder found despite BINGO`)
	td.Cmp(t, e.Unwrap(), internal.NoResponderFound)
}
