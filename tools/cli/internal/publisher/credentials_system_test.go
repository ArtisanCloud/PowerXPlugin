//go:build darwin || linux

package publisher

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// An explicit opt-in uses one fresh account and a synthetic token. It neither
// enumerates nor modifies existing credentials, and never prints secret bytes.
func TestPublisherSystemCredentialLifecycle(t *testing.T) {
	if os.Getenv("PX_PUBLISHER_SYSTEM_KEYRING") != "1" {
		t.Skip("PX_PUBLISHER_SYSTEM_KEYRING=1")
	}
	id := uuid.NewString()
	random, err := secret()
	if err != nil {
		t.Fatal("synthetic_credential_generation")
	}
	token := "pxpub_" + random
	store := systemCredentials{}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		_ = store.Delete(cleanup, id)
	})
	if err = store.Set(ctx, id, token); err != nil {
		t.Fatal("system_credential_store_failed")
	}
	read, err := store.Get(ctx, id)
	if err != nil || read != token {
		t.Fatal("system_credential_read_failed")
	}
	if err = store.Delete(ctx, id); err != nil {
		t.Fatal("system_credential_delete_failed")
	}
	if _, err = store.Get(ctx, id); err == nil {
		t.Fatal("system_credential_remained")
	}
	t.Log("synthetic_system_credential_write_read_delete_passed")
}
