package database

import (
	"context"
	"testing"
)

func TestOpenStoreRequiresDSN(t *testing.T) {
	store, err := OpenStore(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty POSTGRES_DSN, got nil")
	}
	if store != nil {
		t.Fatal("expected nil store on error")
	}
}

func TestStoreCloseNilSafe(t *testing.T) {
	var store *Store
	store.Close()

	store = &Store{}
	store.Close()
}
