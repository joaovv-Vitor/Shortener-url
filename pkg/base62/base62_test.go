package base62

import (
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "A"},
		{35, "Z"},
		{36, "a"},
		{61, "z"},
		{62, "10"},
		{1234, "Ju"},
		{999999, "4C91"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := Encode(tt.input)
			if result != tt.expected {
				t.Errorf("Encode(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"0", 0},
		{"1", 1},
		{"A", 10},
		{"z", 61},
		{"10", 62},
		{"Ju", 1234},
		{"4C91", 999999},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Decode(tt.input)
			if err != nil {
				t.Fatalf("Decode(%q) returned error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("Decode(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	values := []uint64{0, 1, 62, 1000, 123456789, 9999999999}
	for _, v := range values {
		encoded := Encode(v)
		decoded, err := Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(Encode(%d)) returned error: %v", v, err)
		}
		if decoded != v {
			t.Errorf("Roundtrip failed: %d -> %q -> %d", v, encoded, decoded)
		}
	}
}

func TestDecodeInvalidCharacter(t *testing.T) {
	_, err := Decode("abc!def")
	if err != ErrInvalidCharacter {
		t.Errorf("expected ErrInvalidCharacter, got %v", err)
	}
}

func TestDecodeEmpty(t *testing.T) {
	_, err := Decode("")
	if err != ErrInvalidCharacter {
		t.Errorf("expected ErrInvalidCharacter, got %v", err)
	}
}
