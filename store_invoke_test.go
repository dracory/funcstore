package funcstore

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFunctionInvokeByID(t *testing.T) {
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
		SetSource("return input * 2")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	out, err := store.FunctionInvoke(context.Background(), fn.GetID(), 21)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	got, ok := out.(int64)
	if !ok {
		t.Fatalf("FunctionInvoke() result type = %T, want int64", out)
	}
	if got != 42 {
		t.Errorf("FunctionInvoke() result = %d, want %d", got, 42)
	}
}

func TestFunctionInvokeByAlias(t *testing.T) {
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
		SetSource("return input + ' world'").
		SetAlias("greet")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	out, err := store.FunctionInvoke(context.Background(), "greet", "hello")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	got, ok := out.(string)
	if !ok {
		t.Fatalf("FunctionInvoke() result type = %T, want string", out)
	}
	if got != "hello world" {
		t.Errorf("FunctionInvoke() result = %q, want %q", got, "hello world")
	}
}

func TestFunctionInvokeNotFound(t *testing.T) {
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

	_, err = store.FunctionInvoke(context.Background(), "nonexistent", nil)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("FunctionInvoke() non-existent error = %v, want ErrNotFound", err)
	}
}

func TestFunctionInvokeEmptyKey(t *testing.T) {
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

	_, err = store.FunctionInvoke(context.Background(), "", nil)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("FunctionInvoke() empty key error = %v, want ErrInvalidID", err)
	}
}

func TestFunctionCreateUnknownRuntime(t *testing.T) {
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

	err = store.FunctionCreate(context.Background(), NewFunction().
		SetRuntime("lua").
		SetSource("print('hi')"))
	if !errors.Is(err, ErrUnknownRuntime) {
		t.Errorf("FunctionCreate() unknown runtime error = %v, want ErrUnknownRuntime", err)
	}
}

func TestFunctionCreateMissingRuntime(t *testing.T) {
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

	err = store.FunctionCreate(context.Background(), NewFunction().SetSource("return 1"))
	if err == nil {
		t.Fatal("FunctionCreate() with empty Runtime: expected error, got nil")
	}
}

func TestFunctionCreateMissingSource(t *testing.T) {
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

	err = store.FunctionCreate(context.Background(), NewFunction().SetRuntime("js"))
	if err == nil {
		t.Fatal("FunctionCreate() with empty Source: expected error, got nil")
	}
}

func TestFunctionCreateSyntaxError(t *testing.T) {
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

	err = store.FunctionCreate(context.Background(), NewFunction().
		SetRuntime("js").
		SetSource("var ++++ = ;;;"))
	if err == nil {
		t.Fatal("FunctionCreate() with invalid syntax: expected error, got nil")
	}
}

func TestFunctionCreateAliasTaken(t *testing.T) {
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

	err = store.FunctionCreate(context.Background(), NewFunction().
		SetRuntime("js").
		SetSource("return 1").
		SetAlias("dup"))
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	err = store.FunctionCreate(context.Background(), NewFunction().
		SetRuntime("js").
		SetSource("return 2").
		SetAlias("dup"))
	if !errors.Is(err, ErrAliasTaken) {
		t.Errorf("FunctionCreate() duplicate alias error = %v, want ErrAliasTaken", err)
	}
}

func TestFunctionInvokeTimeout(t *testing.T) {
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
		SetSource("var end = Date.now() + 5000; while(Date.now() < end) {}; return 'done'").
		SetTimeoutMs(50)
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	_, err = store.FunctionInvoke(context.Background(), fn.GetID(), nil)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("FunctionInvoke() timeout error = %v, want ErrTimeout", err)
	}
}

func TestFunctionInvokeRuntimeError(t *testing.T) {
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
		SetSource("throw new Error('boom')")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	_, err = store.FunctionInvoke(context.Background(), fn.GetID(), nil)
	if !errors.Is(err, ErrRuntime) {
		t.Errorf("FunctionInvoke() runtime error = %v, want ErrRuntime", err)
	}
}

func TestFunctionInvokeAliasFallback(t *testing.T) {
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
		SetSource("return input").
		SetAlias("fb")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	out1, err := store.FunctionInvoke(context.Background(), fn.GetID(), "a")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if out1.(string) != "a" {
		t.Errorf("FunctionInvoke() by ID = %q, want %q", out1, "a")
	}

	out2, err := store.FunctionInvoke(context.Background(), "fb", "b")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if out2.(string) != "b" {
		t.Errorf("FunctionInvoke() by alias = %q, want %q", out2, "b")
	}
}

func TestFunctionInvokeWithObjectInput(t *testing.T) {
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
		SetSource("return input.name + ' is ' + input.age")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	out, err := store.FunctionInvoke(context.Background(), fn.GetID(), map[string]any{
		"name": "Alice",
		"age":  30,
	})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	got, ok := out.(string)
	if !ok {
		t.Fatalf("FunctionInvoke() result type = %T, want string", out)
	}
	if got != "Alice is 30" {
		t.Errorf("FunctionInvoke() result = %q, want %q", got, "Alice is 30")
	}
}

func TestFunctionInvokeDefaultTimeout(t *testing.T) {
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
		SetSource("return 'fast'")
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatal("unexpected error:", err)
	}

	start := time.Now()
	out, err := store.FunctionInvoke(context.Background(), fn.GetID(), nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if out.(string) != "fast" {
		t.Errorf("FunctionInvoke() result = %q, want %q", out, "fast")
	}
	if elapsed > time.Second {
		t.Errorf("FunctionInvoke() took %v, want under 1s (default timeout is 5s)", elapsed)
	}
}

func TestFunctionCreateNoRuntimeRegistered(t *testing.T) {
	db := initDB()

	store, err := NewStore(StoreOptions{
		DB:                 db,
		Driver:             "sqlite",
		AutomigrateEnabled: true,
		// No runtimes registered
	})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	err = store.FunctionCreate(context.Background(), NewFunction().
		SetRuntime("js").
		SetSource("return 1"))
	if !errors.Is(err, ErrUnknownRuntime) {
		t.Errorf("FunctionCreate() with no runtime error = %v, want ErrUnknownRuntime", err)
	}
}

func TestRegisterRuntimeAfterConstruction(t *testing.T) {
	db := initDB()

	store, err := NewStore(StoreOptions{
		DB:                 db,
		Driver:             "sqlite",
		AutomigrateEnabled: true,
		// No runtimes registered initially
	})
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	// Register a runtime after construction.
	storeSQL := store.(*storeSQL)
	storeSQL.RegisterRuntime(NewGojaRuntime())

	err = store.FunctionCreate(context.Background(), NewFunction().
		SetRuntime("js").
		SetSource("return 1"))
	if err != nil {
		t.Fatalf("FunctionCreate() after RegisterRuntime: unexpected error: %v", err)
	}
}
