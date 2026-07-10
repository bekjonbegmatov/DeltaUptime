package app

import (
	"context"
	"strings"
	"testing"
)

func TestRun_Version(t *testing.T) {
	var out strings.Builder
	if err := Run(context.Background(), []string{"version"}, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != Version {
		t.Fatalf("version output = %q, want %q", got, Version)
	}
}

func TestRun_NoArgsPrintsUsage(t *testing.T) {
	var out strings.Builder
	if err := Run(context.Background(), nil, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected usage text, got: %q", out.String())
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var out strings.Builder
	err := Run(context.Background(), []string{"frobnicate"}, &out)
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Fatalf("error should mention the command, got: %v", err)
	}
}

func TestRun_MigrateWithoutDSN(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "") // no database configured

	var out strings.Builder
	err := Run(context.Background(), []string{"migrate"}, &out)
	if err == nil {
		t.Fatal("expected error when POSTGRES_DSN is empty, got nil")
	}
	if !strings.Contains(err.Error(), "POSTGRES_DSN") {
		t.Fatalf("error should mention POSTGRES_DSN, got: %v", err)
	}
}

func TestRun_PlaceholderStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: placeholder must return immediately

	var out strings.Builder
	if err := Run(ctx, []string{"scheduler"}, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
