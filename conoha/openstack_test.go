package conoha

import (
	"testing"
)

func TestNewOpenStack(t *testing.T) {
	os, err := NewOpenStack()
	if err != nil || os == nil {
		t.Errorf("%v", err)
	}

	if os.Compute == nil {
		t.Fatal("os.Compute should not be nil")
	}
	if os.Network == nil {
		t.Fatal("os.Network should not be nil")
	}
}
