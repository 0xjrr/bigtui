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
