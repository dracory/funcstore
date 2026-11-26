package funcstore

import (
	"context"
	"errors"
	"testing"
)

func TestVersionAutoSnapshotOnSourceChange(t *testing.T) {
	store := newTestStore(t)

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	// No versions yet.
	versions, err := store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 0 {
		t.Errorf("initial version count = %d, want 0", len(versions))
	}

	// Update source → should snapshot the old source.
	fn.SetSource("return 2")
	if err := store.FunctionUpdate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	versions, err = store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 1 {
		t.Fatalf("version count after source change = %d, want 1", len(versions))
	}
	if versions[0].GetSource() != "return 1" {
		t.Errorf("snapshot source = %q, want %q", versions[0].GetSource(), "return 1")
	}
}

func TestVersionNoSnapshotOnSettingsOnlyChange(t *testing.T) {
	store := newTestStore(t)

	fn := NewFunction().SetRuntime("js").SetSource("return 1").SetName("original")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	// Change only the name, not the source.
	fn.SetName("renamed")
	if err := store.FunctionUpdate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	versions, err := store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 0 {
		t.Errorf("version count after settings-only change = %d, want 0", len(versions))
	}
}

func TestVersionMultipleSnapshots(t *testing.T) {
	store := newTestStore(t)

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	for i := 2; i <= 4; i++ {
		fn.SetSource("return " + string(rune('0'+i)))
		if err := store.FunctionUpdate(context.Background(), fn); err != nil {
			t.Fatal("unexpected error:", err)
		}
	}

	versions, err := store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 3 {
		t.Errorf("version count = %d, want 3", len(versions))
	}
}

func TestVersionGet(t *testing.T) {
	store := newTestStore(t)

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	fn.SetSource("return 2")
	if err := store.FunctionUpdate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	versions, err := store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 1 {
		t.Fatalf("version count = %d, want 1", len(versions))
	}

	v, err := store.FunctionVersionGet(context.Background(), versions[0].GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if v.GetSource() != "return 1" {
		t.Errorf("Get() source = %q, want %q", v.GetSource(), "return 1")
	}
	if v.GetFunctionID() != fn.GetID() {
		t.Errorf("Get() function_id = %q, want %q", v.GetFunctionID(), fn.GetID())
	}
}

func TestVersionGetNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.FunctionVersionGet(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() non-existent error = %v, want ErrNotFound", err)
	}
}

func TestVersionDelete(t *testing.T) {
	store := newTestStore(t)

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	fn.SetSource("return 2")
	if err := store.FunctionUpdate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	versions, err := store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 1 {
		t.Fatalf("version count = %d, want 1", len(versions))
	}

	if err := store.FunctionVersionDelete(context.Background(), versions[0].GetID()); err != nil {
		t.Fatal("unexpected error:", err)
	}

	versions, err = store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 0 {
		t.Errorf("version count after delete = %d, want 0", len(versions))
	}
}

func TestVersionDeleteNotFound(t *testing.T) {
	store := newTestStore(t)

	err := store.FunctionVersionDelete(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() non-existent error = %v, want ErrNotFound", err)
	}
}

func TestFunctionRestore(t *testing.T) {
	store := newTestStore(t)

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	// Change source twice to create two versions.
	fn.SetSource("return 2")
	if err := store.FunctionUpdate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}
	fn.SetSource("return 3")
	if err := store.FunctionUpdate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	// Current source should be "return 3".
	found, err := store.FunctionFindByID(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if found.GetSource() != "return 3" {
		t.Fatalf("current source = %q, want %q", found.GetSource(), "return 3")
	}

	// List versions: newest first → ["return 2", "return 1"]
	versions, err := store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 2 {
		t.Fatalf("version count = %d, want 2", len(versions))
	}

	// Restore the oldest version ("return 1").
	if err := store.FunctionRestore(context.Background(), fn.GetID(), versions[1].GetID()); err != nil {
		t.Fatal("unexpected error:", err)
	}

	// Source should now be "return 1".
	found, err = store.FunctionFindByID(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if found.GetSource() != "return 1" {
		t.Errorf("after Restore() source = %q, want %q", found.GetSource(), "return 1")
	}

	// Restore should have snapshotted "return 3" → now 3 versions.
	versions, err = store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 3 {
		t.Errorf("version count after restore = %d, want 3", len(versions))
	}

	// The newest version should be the pre-restore source ("return 3").
	if versions[0].GetSource() != "return 3" {
		t.Errorf("newest version source = %q, want %q", versions[0].GetSource(), "return 3")
	}
}

func TestFunctionRestoreWrongFunctionID(t *testing.T) {
	store := newTestStore(t)

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	fn.SetSource("return 2")
	if err := store.FunctionUpdate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	versions, err := store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	// Try to restore with a wrong function ID.
	err = store.FunctionRestore(context.Background(), "wrong-id", versions[0].GetID())
	if err == nil {
		t.Error("Restore() with wrong function ID should return error")
	}
}

func TestVersionListEmpty(t *testing.T) {
	store := newTestStore(t)

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	versions, err := store.FunctionVersionList(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(versions) != 0 {
		t.Errorf("version count = %d, want 0", len(versions))
	}
}

// newTestStore creates a fresh in-memory store for version tests.
func newTestStore(t *testing.T) StoreInterface {
	t.Helper()
	db := initDB()
	store, err := NewStore(StoreOptions{
		DB:                 db,
		Driver:             "sqlite",
		AutomigrateEnabled: true,
		Runtimes:           []Runtime{NewGojaRuntime()},
	})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	return store
}
