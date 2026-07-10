package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestEmbeddedMigrationsPresent(t *testing.T) {
	entries, err := fs.Glob(FS, "*.sql")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no *.sql migrations embedded")
	}
}

// Every migration must declare both Up and Down sections so it is reversible.
func TestMigrationsHaveUpAndDown(t *testing.T) {
	entries, _ := fs.Glob(FS, "*.sql")
	for _, name := range entries {
		b, err := fs.ReadFile(FS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		body := string(b)
		if !strings.Contains(body, "-- +goose Up") {
			t.Errorf("%s: missing '-- +goose Up'", name)
		}
		if !strings.Contains(body, "-- +goose Down") {
			t.Errorf("%s: missing '-- +goose Down'", name)
		}
	}
}
