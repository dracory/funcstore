package funcstore

import (
	"context"
	"errors"
	goruntime "runtime"
	"testing"
	"time"
)

func TestTimeoutInterruptsExecution(t *testing.T) {
	store, err := NewStore(StoreOptions{DB: initDB(), Driver: "sqlite", Runtimes: []Runtime{NewGojaRuntime()}, AutomigrateEnabled: true})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	fn := NewFunction().
		SetRuntime("js").
		SetSource("while(true) {}").
		SetTimeoutMs(50)
	if err := store.FunctionCreate(context.Background(), fn); err != nil {
		t.Fatalf("create: %v", err)
	}

	before := goruntime.NumGoroutine()
	_, err = store.FunctionInvoke(context.Background(), fn.GetID(), nil)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want ErrTimeout", err)
	}

	// Give the interrupted goroutine time to unwind. Without the interrupt
	// it would spin forever, so goroutine count would stay elevated.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if goruntime.NumGoroutine() <= before+1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("goroutine still running after interrupt: before=%d now=%d", before, goruntime.NumGoroutine())
}
