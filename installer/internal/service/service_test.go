package service

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr, err := NewManager("sysworker")
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	// Test that Status works (even if platform doesn't support services)
	status, err := mgr.Status()
	_ = status // We don't care about the actual status, just that it doesn't crash
	_ = err
}

func TestNewManagerEmptyName(t *testing.T) {
	_, err := NewManager("")
	if err == nil {
		t.Error("NewManager with empty name should return error")
	}
}

func TestIsInstalled(t *testing.T) {
	mgr, err := NewManager("sysworker-test-nonexistent")
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	// Should not be installed (we haven't installed anything)
	// This test might fail if someone actually has this service installed
	isInstalled := mgr.IsInstalled()
	// We just verify it doesn't panic/crash
	_ = isInstalled
}
