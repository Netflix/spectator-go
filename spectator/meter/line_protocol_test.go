package meter

import (
	"math"
	"testing"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

func TestWriteLineInt(t *testing.T) {
	tests := map[string]struct {
		symbol   string
		spectId  string
		val      int64
		expected string
	}{
		"zero":      {symbol: "c", spectId: "name", val: 0, expected: "c:name:0"},
		"positive":  {symbol: "c", spectId: "name", val: 42, expected: "c:name:42"},
		"negative":  {symbol: "c", spectId: "name", val: -1, expected: "c:name:-1"},
		"max_int64": {symbol: "c", spectId: "name", val: math.MaxInt64, expected: "c:name:9223372036854775807"},
		"min_int64": {symbol: "c", spectId: "name", val: math.MinInt64, expected: "c:name:-9223372036854775808"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			w := writer.MemoryWriter{}
			writeLineInt(&w, tt.symbol, tt.spectId, tt.val)
			if got := w.Lines()[0]; got != tt.expected {
				t.Errorf("Expected line to be %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestWriteLineFloat(t *testing.T) {
	tests := map[string]struct {
		symbol   string
		spectId  string
		val      float64
		expected string
	}{
		"zero":                {symbol: "g", spectId: "name", val: 0.0, expected: "g:name:0.000000"},
		"positive":            {symbol: "g", spectId: "name", val: 100.1, expected: "g:name:100.100000"},
		"negative":            {symbol: "g", spectId: "name", val: -100.1, expected: "g:name:-100.100000"},
		"tiny_rounds_to_zero": {symbol: "g", spectId: "name", val: 1e-7, expected: "g:name:0.000000"},
		"large":               {symbol: "g", spectId: "name", val: 1e15, expected: "g:name:1000000000000000.000000"},
		"nan":                 {symbol: "g", spectId: "name", val: math.NaN(), expected: "g:name:NaN"},
		"positive_inf":        {symbol: "g", spectId: "name", val: math.Inf(1), expected: "g:name:+Inf"},
		"negative_inf":        {symbol: "g", spectId: "name", val: math.Inf(-1), expected: "g:name:-Inf"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			w := writer.MemoryWriter{}
			writeLineFloat(&w, tt.symbol, tt.spectId, tt.val)
			if got := w.Lines()[0]; got != tt.expected {
				t.Errorf("Expected line to be %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestWriteLineUint(t *testing.T) {
	tests := map[string]struct {
		symbol   string
		spectId  string
		val      uint64
		expected string
	}{
		"zero":       {symbol: "U", spectId: "name", val: 0, expected: "U:name:0"},
		"positive":   {symbol: "U", spectId: "name", val: 42, expected: "U:name:42"},
		"max_uint64": {symbol: "U", spectId: "name", val: math.MaxUint64, expected: "U:name:18446744073709551615"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			w := writer.MemoryWriter{}
			writeLineUint(&w, tt.symbol, tt.spectId, tt.val)
			if got := w.Lines()[0]; got != tt.expected {
				t.Errorf("Expected line to be %s, got %s", tt.expected, got)
			}
		})
	}
}
