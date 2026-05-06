package invalidation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bool64/brick/database"
	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	_ "modernc.org/sqlite"
)

type stubDeleter struct {
	deleted []string
}

func (s *stubDeleter) Delete(_ context.Context, key []byte) error {
	s.deleted = append(s.deleted, string(key))

	return nil
}

func TestIndexInvalidateByLabels(t *testing.T) {
	st := testStorage(t)
	idx := NewIndex(st)

	main := &stubDeleter{}
	other := &stubDeleter{}
	idx.AddCache("main-page", main)
	idx.AddCache("album-data", other)

	idx.AddLabels("main-page", []byte("k1"), "album/a", "service-settings")
	idx.AddLabels("main-page", []byte("k2"), "album/b")
	idx.AddLabels("album-data", []byte("k3"), "album/a")

	n, err := idx.InvalidateByLabels(context.Background(), "album/a")
	if err != nil {
		t.Fatalf("invalidate by labels: %v", err)
	}

	if n != 2 {
		t.Fatalf("unexpected delete count: %d", n)
	}

	if len(main.deleted) != 1 || main.deleted[0] != "k1" {
		t.Fatalf("unexpected main deletions: %#v", main.deleted)
	}

	if len(other.deleted) != 1 || other.deleted[0] != "k3" {
		t.Fatalf("unexpected album deletions: %#v", other.deleted)
	}

	n, err = idx.InvalidateByLabels(context.Background(), "album/a")
	if err != nil {
		t.Fatalf("second invalidate by labels: %v", err)
	}

	if n != 0 {
		t.Fatalf("unexpected second delete count: %d", n)
	}
}

func TestIndexResetKey(t *testing.T) {
	st := testStorage(t)
	idx := NewIndex(st)

	d := &stubDeleter{}
	idx.AddCache("album-data", d)
	idx.AddLabels("album-data", []byte("preview"), "album/a", "album/b")

	if err := idx.ResetKey(context.Background(), "album-data", []byte("preview")); err != nil {
		t.Fatalf("reset key: %v", err)
	}

	n, err := idx.InvalidateByLabels(context.Background(), "album/a", "album/b")
	if err != nil {
		t.Fatalf("invalidate after reset key: %v", err)
	}

	if n != 0 {
		t.Fatalf("unexpected delete count after reset key: %d", n)
	}

	if len(d.deleted) != 0 {
		t.Fatalf("unexpected deletions after reset key: %#v", d.deleted)
	}
}

func testStorage(t *testing.T) *sqluct.Storage {
	t.Helper()

	cfg := database.Config{
		DriverName:      "sqlite",
		DSN:             filepath.Join(t.TempDir(), "invalidation.sqlite") + "?_time_format=sqlite",
		ApplyMigrations: true,
		MaxOpen:         1,
		MaxIdle:         1,
	}

	st, err := database.SetupStorageDSN(cfg, ctxd.NoOpLogger{}, stats.NoOp{}, Migrations)
	if err != nil {
		t.Fatalf("setup storage: %v", err)
	}

	t.Cleanup(func() {
		if err := st.DB().DB.Close(); err != nil {
			t.Fatalf("close storage: %v", err)
		}
	})

	return st
}

var _ cache.Deleter = (*stubDeleter)(nil)
