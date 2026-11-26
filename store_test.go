package funcstore

import (
	"context"
	"errors"
	"testing"
)

func TestStoreCreate(t *testing.T) {
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

	fn := NewFunction().
		SetRuntime("js").
		SetSource("return 1").
		SetAlias("test-fn")

	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if fn.GetID() == "" {
		t.Error("Create() must set a non-empty ID")
	}
	if fn.GetCreatedAt() == "" {
		t.Error("Create() must set CreatedAt")
	}
	if fn.GetUpdatedAt() == "" {
		t.Error("Create() must set UpdatedAt")
	}
}

func TestStoreCreateAutoGeneratesID(t *testing.T) {
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

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	// Clear the auto-generated ID from NewFunction to test store auto-generation.
	fn.SetID("")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}
	if fn.GetID() == "" {
		t.Error("Create() must auto-generate ID when empty")
	}
}

func TestStoreFindByID(t *testing.T) {
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

	fn := NewFunction().
		SetRuntime("js").
		SetSource("return input * 2").
		SetAlias("double").
		SetName("Doubler")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	found, err := store.FunctionFindByID(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if found == nil {
		t.Fatal("FindByID() returned nil, want non-nil")
	}
	if found.GetSource() != "return input * 2" {
		t.Errorf("FindByID().GetSource() = %q, want %q", found.GetSource(), "return input * 2")
	}
	if found.GetAlias() != "double" {
		t.Errorf("FindByID().GetAlias() = %q, want %q", found.GetAlias(), "double")
	}
	if found.GetName() != "Doubler" {
		t.Errorf("FindByID().GetName() = %q, want %q", found.GetName(), "Doubler")
	}
}

func TestStoreFindByIDNotFound(t *testing.T) {
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

	found, err := store.FunctionFindByID(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("FindByID() non-existent error = %v, want ErrNotFound", err)
	}
	if found != nil {
		t.Error("FindByID() non-existent should return nil")
	}
}

func TestStoreFindByIDEmpty(t *testing.T) {
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

	_, err = store.FunctionFindByID(context.Background(), "")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("FindByID() empty error = %v, want ErrInvalidID", err)
	}
}

func TestStoreFindByAlias(t *testing.T) {
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

	fn := NewFunction().SetRuntime("js").SetSource("return 1").SetAlias("greet")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	found, err := store.FunctionFindByAlias(context.Background(), "greet")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if found == nil {
		t.Fatal("FindByAlias() returned nil, want non-nil")
	}
	if found.GetID() != fn.GetID() {
		t.Errorf("FindByAlias().GetID() = %q, want %q", found.GetID(), fn.GetID())
	}
}

func TestStoreFindByAliasNotFound(t *testing.T) {
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

	found, err := store.FunctionFindByAlias(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("FindByAlias() non-existent error = %v, want ErrNotFound", err)
	}
	if found != nil {
		t.Error("FindByAlias() non-existent should return nil")
	}
}

func TestStoreAliasUniqueness(t *testing.T) {
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

	fn1 := NewFunction().SetRuntime("js").SetSource("return 1").SetAlias("dup")
	if err := store.FunctionCreate(context.Background(), fn1); err != nil {
		t.Fatal("unexpected error:", err)
	}

	fn2 := NewFunction().SetRuntime("js").SetSource("return 2").SetAlias("dup")
	err = store.FunctionCreate(context.Background(), fn2)
	if !errors.Is(err, ErrAliasTaken) {
		t.Errorf("Create() duplicate alias error = %v, want ErrAliasTaken", err)
	}
}

func TestStoreEmptyAliasAllowed(t *testing.T) {
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

	fn1 := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn1); err != nil {
		t.Fatal("unexpected error:", err)
	}

	fn2 := NewFunction().SetRuntime("js").SetSource("return 2")
	if err := store.FunctionCreate(context.Background(), fn2); err != nil {
		t.Fatal("unexpected error:", err)
	}
}

func TestStoreList(t *testing.T) {
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

	for i := 0; i < 3; i++ {
		fn := NewFunction().SetRuntime("js").SetSource("return input")
		if err := store.FunctionCreate(context.Background(), fn); err != nil {
			t.Fatal("unexpected error:", err)
		}
	}

	list, err := store.FunctionList(context.Background())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(list) != 3 {
		t.Errorf("List() len = %d, want %d", len(list), 3)
	}
}

func TestStoreListEmpty(t *testing.T) {
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

	list, err := store.FunctionList(context.Background())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(list) != 0 {
		t.Errorf("List() empty len = %d, want 0", len(list))
	}
}

func TestStoreUpdate(t *testing.T) {
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

	fn := NewFunction().SetRuntime("js").SetSource("return input").SetName("original")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	fn.SetSource("return input * 10").SetName("updated")
	if err := store.FunctionUpdate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	found, err := store.FunctionFindByID(context.Background(), fn.GetID())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if found.GetSource() != "return input * 10" {
		t.Errorf("after Update() GetSource() = %q, want %q", found.GetSource(), "return input * 10")
	}
	if found.GetName() != "updated" {
		t.Errorf("after Update() GetName() = %q, want %q", found.GetName(), "updated")
	}
}

func TestStoreUpdateNotFound(t *testing.T) {
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

	err = store.FunctionUpdate(context.Background(), NewFunction().
		SetID("nonexistent").
		SetRuntime("js").
		SetSource("return 1"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Update() non-existent error = %v, want ErrNotFound", err)
	}
}

func TestStoreDelete(t *testing.T) {
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

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	if err := store.FunctionDelete(context.Background(), fn.GetID()); err != nil {
		t.Fatal("unexpected error:", err)
	}

	found, err := store.FunctionFindByID(context.Background(), fn.GetID())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("after Delete() FindByID() error = %v, want ErrNotFound", err)
	}
	if found != nil {
		t.Error("after Delete() FindByID() should return nil")
	}
}

func TestStoreDeleteNotFound(t *testing.T) {
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

	err = store.FunctionDelete(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() non-existent error = %v, want ErrNotFound", err)
	}
}

func TestStoreDeleteEmptyID(t *testing.T) {
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

	err = store.FunctionDelete(context.Background(), "")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("Delete() empty error = %v, want ErrInvalidID", err)
	}
}

func TestStoreDB(t *testing.T) {
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

	if store.DB() != db {
		t.Error("DB() should return the same *sql.DB passed in")
	}
}

func TestStoreNilDB(t *testing.T) {
	_, err := NewStore(StoreOptions{
		DB:                 nil,
		Driver:             "sqlite",
		AutomigrateEnabled: true,
		Runtimes:           []Runtime{NewGojaRuntime()},
	})
	if err == nil {
		t.Fatal("NewStore() with nil DB: expected error, got nil")
	}
}

func TestStoreCustomTableName(t *testing.T) {
	db := initDB()

	store, err := NewStore(StoreOptions{
		DB:                 db,
		Driver:             "sqlite",
		TableName:          "my_functions",
		AutomigrateEnabled: true,
		Runtimes:           []Runtime{NewGojaRuntime()},
	})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	fn := NewFunction().SetRuntime("js").SetSource("return 1")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM my_functions").Scan(&count)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if count != 1 {
		t.Errorf("custom table row count = %d, want 1", count)
	}
}

func TestStoreMigrateUp(t *testing.T) {
	db := initDB()

	store, err := NewStore(StoreOptions{
		DB:     db,
		Driver: "sqlite",
	})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='funcstore_functions'").Scan(&tableName)
	if err == nil {
		t.Fatal("table should not exist before MigrateUp")
	}

	if err := store.MigrateUp(context.Background()); err != nil {
		t.Fatal("unexpected error:", err)
	}

	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='funcstore_functions'").Scan(&tableName)
	if err != nil {
		t.Fatal("table should exist after MigrateUp")
	}
}

func TestStoreMigrateUpIdempotent(t *testing.T) {
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

	if err := store.MigrateUp(context.Background()); err != nil {
		t.Fatal("unexpected error:", err)
	}
}

func TestStoreMigrateDown(t *testing.T) {
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

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='funcstore_functions'").Scan(&tableName)
	if err != nil {
		t.Fatal("table should exist before MigrateDown")
	}

	if err := store.MigrateDown(context.Background()); err != nil {
		t.Fatal("unexpected error:", err)
	}

	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='funcstore_functions'").Scan(&tableName)
	if err == nil {
		t.Fatal("table should not exist after MigrateDown")
	}
}
