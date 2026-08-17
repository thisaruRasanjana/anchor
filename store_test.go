package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestStoreSetAndGet(t *testing.T) {
	store := NewStore()

	store.Set("name", "Thisaru")

	value, ok := store.Get("name")

	if !ok {
		t.Fatal("expected key to exist")
	}

	if value != "Thisaru" {
		t.Fatalf("expected Thisaru, got %s", value)
	}
}

func TestStoreGetMissingKey(t *testing.T) {
	store := NewStore()

	value, ok := store.Get("missing")

	if ok {
		t.Fatal("expected key not to exist")
	}

	if value != "" {
		t.Fatalf("expected empty value, got %q", value)
	}
}

func TestStoreDelete(t *testing.T) {
	store := NewStore()

	store.Set("name", "Thisaru")

	deleted := store.Delete("name")

	if !deleted {
		t.Fatal("expected Delete to return true")
	}

	_, ok := store.Get("name")

	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestStoreDeleteMissingKey(t *testing.T) {
	store := NewStore()

	deleted := store.Delete("missing")

	if deleted {
		t.Fatal("expected Delete to return false")
	}
}

func TestStoreTTL(t *testing.T) {
	store := NewStore()

	store.SetWithTTL("token", "abc123", 1)

	value, ok := store.Get("token")

	if !ok {
		t.Fatal("expected key to exist before expiration")
	}

	if value != "abc123" {
		t.Fatalf("expected abc123, got %s", value)
	}
}

func TestStoreTTLExpiration(t *testing.T) {
	store := NewStore()

	store.SetWithTTL("token", "abc123", 1)

	time.Sleep(1100 * time.Millisecond)

	_, ok := store.Get("token")

	if ok {
		t.Fatal("expected key to be expired")
	}
}

func TestStoreSetWithExpiration(t *testing.T) {
	store := NewStore()

	expiresAt := time.Now().Add(1 * time.Second)

	store.SetWithExpiration("token", "abc123", expiresAt)

	value, ok := store.Get("token")

	if !ok {
		t.Fatal("expected key to exist")
	}

	if value != "abc123" {
		t.Fatalf("expected abc123, got %s", value)
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	store := NewStore()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			key := fmt.Sprintf("key%d", i)
			value := fmt.Sprintf("value%d", i)

			store.Set(key, value)

			got, ok := store.Get(key)
			if !ok {
				t.Errorf("expected %s to exist", key)
				return
			}

			if got != value {
				t.Errorf("expected %s, got %s", value, got)
			}
		}(i)
	}

	wg.Wait()
}
