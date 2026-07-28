package apperr

import (
	"errors"
	"testing"
)

func TestSentinels(t *testing.T) {
	if !errors.Is(NotFound("x"), ErrNotFound) {
		t.Fatal("NotFound should wrap ErrNotFound")
	}
	if !errors.Is(Invalid("x"), ErrInvalid) {
		t.Fatal("Invalid should wrap ErrInvalid")
	}
}

func TestAsApp(t *testing.T) {
	err := Conflict("dup")
	ae, ok := AsApp(err)
	if !ok || ae.Code != "conflict" {
		t.Fatalf("AsApp = %#v ok=%v", ae, ok)
	}
	if _, ok := AsApp(errors.New("plain")); ok {
		t.Fatal("plain error should not AsApp")
	}
}
