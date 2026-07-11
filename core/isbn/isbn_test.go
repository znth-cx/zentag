package isbn

import "testing"

func TestValidate_ISBN10Valid(t *testing.T) {
	ok, err := Validate("0306406152")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !ok {
		t.Error("Validate() = false, want true")
	}
}

func TestValidate_ISBN10Hyphenated(t *testing.T) {
	ok, err := Validate("0-306-40615-2")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !ok {
		t.Error("Validate() = false, want true")
	}
}

func TestValidate_ISBN10InvalidChecksum(t *testing.T) {
	ok, err := Validate("0306406151")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if ok {
		t.Error("Validate() = true, want false")
	}
}

func TestValidate_ISBN10XCheckDigit(t *testing.T) {
	// 0-9752298-0-X: a real ISBN-10 whose check digit is X (value 10).
	ok, err := Validate("097522980X")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !ok {
		t.Error("Validate() = false, want true")
	}
}

func TestValidate_ISBN13Valid(t *testing.T) {
	ok, err := Validate("9780306406157")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !ok {
		t.Error("Validate() = false, want true")
	}
}

func TestValidate_ISBN13InvalidChecksum(t *testing.T) {
	ok, err := Validate("9780306406158")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if ok {
		t.Error("Validate() = true, want false")
	}
}

func TestValidate_WrongLength(t *testing.T) {
	_, err := Validate("12345")
	if err == nil {
		t.Fatal("Validate() error = nil, want error for wrong length")
	}
}

func TestValidate_NonDigitCharacter(t *testing.T) {
	_, err := Validate("03064A6152")
	if err == nil {
		t.Fatal("Validate() error = nil, want error for non-digit character")
	}
}
