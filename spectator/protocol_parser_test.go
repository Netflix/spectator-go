package spectator

import (
	"testing"
)

func TestParseProtocolLineWithValidInput(t *testing.T) {
	line := "c:meterId,tag1=value1,tag2=value2:value"
	meterType, meterId, value, err := ParseProtocolLine(line)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if meterType != "c" {
		t.Errorf("Expected 'c', got '%s'", meterType)
	}

	if meterId.Name() != "meterId" || meterId.Tags()["tag1"] != "value1" || meterId.Tags()["tag2"] != "value2" {
		t.Errorf("Unexpected meterId: %v", meterId)
	}

	if value != "value" {
		t.Errorf("Expected 'value', got '%s'", value)
	}
}

func TestParseProtocolLineWithInvalidFormat(t *testing.T) {
	line := "invalid_format_line"
	_, _, _, err := ParseProtocolLine(line)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestParseProtocolLineWithInvalidTagFormat(t *testing.T) {
	line := "c:meterId=value=value2,tag1=value1,tag2:value"
	_, _, _, err := ParseProtocolLine(line)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestParseGaugeWithTTL(t *testing.T) {
	line := "g,120:test:1"
	meterSymbol, meterId, value, err := ParseProtocolLine(line)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if meterSymbol != "g,120" {
		t.Errorf("Expected 'g', got '%s'", meterSymbol)
	}

	if meterId.Name() != "test" {
		t.Errorf("Unexpected meterId: %v", meterId)
	}

	if value != "1" {
		t.Errorf("Expected '1', got '%s'", value)
	}
}
