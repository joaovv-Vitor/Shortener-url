package hashids_generator_test

import (
	"testing"

	hashids "github.com/joaovv-Vitor/Shorner-url/pkg/hashids_generator"
)

const testSalt = "test-secret-salt"

func TestEncode_NonSequential(t *testing.T) {
	gen, err := hashids.NewGenerator(testSalt, 6, 9)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	code1, err := gen.Encode(1)
	if err != nil {
		t.Fatalf("Encode(1) failed: %v", err)
	}

	code2, err := gen.Encode(2)
	if err != nil {
		t.Fatalf("Encode(2) failed: %v", err)
	}

	t.Logf("1 → %q, 2 → %q", code1, code2)

	if code1 == code2 {
		t.Error("expected different codes for different IDs")
	}
}

func TestEncode_MinLength(t *testing.T) {
	gen, err := hashids.NewGenerator(testSalt, 6, 9)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	code, err := gen.Encode(1)
	if err != nil {
		t.Fatalf("Encode(1) failed: %v", err)
	}

	if len(code) < 6 {
		t.Errorf("expected at least 6 chars, got %d (%q)", len(code), code)
	}

	if len(code) > 9 {
		t.Errorf("expected at most 9 chars, got %d (%q)", len(code), code)
	}
}

func TestRoundTrip(t *testing.T) {
	gen, err := hashids.NewGenerator(testSalt, 6, 9)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	values := []uint64{1, 2, 100, 1000, 123456, 999999}
	for _, v := range values {
		code, err := gen.Encode(v)
		if err != nil {
			t.Fatalf("Encode(%d) failed: %v", v, err)
		}

		decoded, err := gen.Decode(code)
		if err != nil {
			t.Fatalf("Decode(%q) failed: %v", code, err)
		}

		if decoded != v {
			t.Errorf("Roundtrip failed: %d → %q → %d", v, code, decoded)
		}
	}
}

func TestDifferentSalts_DifferentCodes(t *testing.T) {
	gen1, _ := hashids.NewGenerator("salt-one", 6, 9)
	gen2, _ := hashids.NewGenerator("salt-two", 6, 9)

	code1, _ := gen1.Encode(42)
	code2, _ := gen2.Encode(42)

	if code1 == code2 {
		t.Errorf("different salts should produce different codes, got %q for both", code1)
	}
}

func TestDecode_EmptyCode(t *testing.T) {
	gen, _ := hashids.NewGenerator(testSalt, 6, 9)

	_, err := gen.Decode("")
	if err != hashids.ErrEmptyCode {
		t.Errorf("expected ErrEmptyCode, got %v", err)
	}
}

func TestEncode_UniqueOutputs(t *testing.T) {
	gen, err := hashids.NewGenerator(testSalt, 6, 9)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	seen := make(map[string]uint64)
	for i := uint64(1); i <= 1000; i++ {
		code, err := gen.Encode(i)
		if err != nil {
			t.Fatalf("Encode(%d) failed: %v", i, err)
		}

		if prev, exists := seen[code]; exists {
			t.Fatalf("collision: Encode(%d) and Encode(%d) both produce %q", prev, i, code)
		}
		seen[code] = i
	}
}
