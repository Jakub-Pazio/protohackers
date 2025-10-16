package main

import (
	"testing"
)

func TestParseFailure(t *testing.T) {
	illegalMethod := "glOrp"

	_, err := ParseMethod(illegalMethod)

	if err == nil {
		t.Errorf("Expected error")
	}

	want := "ERR illegal method: glOrp"
	got := err.Error()

	if got != want {
		t.Errorf("want %q, got %q\n", want, got)
	}
}

func TestParse(t *testing.T) {
	method := "Get"

	m, err := ParseMethod(method)

	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	want := GetMethod

	if m != want {
		t.Errorf("want %q, got %q\n", want, m)
	}
}
