package processlock

import (
	"strings"
	"testing"
)

func TestAcquireExcludesSecondProcessLock(t *testing.T) {
	dir := t.TempDir()
	first, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()

	second, err := Acquire(dir)
	if err == nil {
		_ = second.Release()
		t.Fatal("second acquire unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "already owned by another process") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseAllowsReacquire(t *testing.T) {
	dir := t.TempDir()
	first, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}

	second, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Release(); err != nil {
		t.Fatal(err)
	}
}
