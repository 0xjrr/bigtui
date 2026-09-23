package bigquery

import (
	"math/big"
	"testing"
)

func TestFormatValueRendersNumericAsDecimal(t *testing.T) {
	value := new(big.Rat).SetFrac64(4001, 2)
	if got := formatValue(value); got != "2000.5" {
		t.Fatalf("formatValue(%s) = %q, want %q", value, got, "2000.5")
	}
}

func TestFormatValueRendersNumericValuesAndScalars(t *testing.T) {
	cases := map[string]struct {
		value any
		want  string
	}{
		"rat value":   {value: *new(big.Rat).SetFrac64(1, 4), want: "0.25"},
		"whole rat":   {value: new(big.Rat).SetFrac64(4, 2), want: "2"},
		"integer":     {value: int64(42), want: "42"},
		"string":      {value: "orders", want: "orders"},
		"boolean":     {value: true, want: "true"},
		"null value":  {value: nil, want: "<nil>"},
		"float value": {value: 1.5, want: "1.5"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			if got := formatValue(testCase.value); got != testCase.want {
				t.Fatalf("formatValue(%v) = %q, want %q", testCase.value, got, testCase.want)
			}
		})
	}
}

func TestCloudClientSupportsDryRunAnalysis(t *testing.T) {
	var client Client = &CloudClient{}
	if _, ok := client.(Analyzer); !ok {
		t.Fatal("the cloud client should support dry-run analysis")
	}
}
