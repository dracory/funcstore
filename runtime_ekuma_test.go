package funcstore

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestEkumaRuntimeName(t *testing.T) {
	r := NewEkumaRuntime()

	if got := r.Name(); got != "js-ekuma" {
		t.Errorf("Name() = %q, want %q", got, "js-ekuma")
	}
}

func TestEkumaRuntimeCompileAndRun(t *testing.T) {
	r := NewEkumaRuntime()

	prog, err := r.Compile("return input + 1")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(41)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	got, ok := out.(int64)
	if !ok {
		t.Fatalf("Run() result type = %T, want int64", out)
	}
	if got != 42 {
		t.Errorf("Run() result = %d, want %d", got, 42)
	}
}

func TestEkumaRuntimeCompileSyntaxError(t *testing.T) {
	r := NewEkumaRuntime()

	_, err := r.Compile("var ++++ = ;;;")
	if err == nil {
		t.Fatal("Compile() with invalid syntax: expected error, got nil")
	}
	if !errors.Is(err, ErrRuntime) {
		t.Errorf("Compile() error = %v, want ErrRuntime", err)
	}
}

func TestEkumaRuntimeRunThrowsError(t *testing.T) {
	r := NewEkumaRuntime()

	prog, err := r.Compile("throw new Error('boom')")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	_, err = prog.Run(nil)
	if err == nil {
		t.Fatal("Run() with throw: expected error, got nil")
	}
	if !errors.Is(err, ErrRuntime) {
		t.Errorf("Run() error = %v, want ErrRuntime", err)
	}
}

func TestEkumaRuntimeRunWithObjectInput(t *testing.T) {
	r := NewEkumaRuntime()

	prog, err := r.Compile("return input.name + ' is ' + input.age")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{"name": "Alice", "age": 30})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got, ok := out.(string); !ok || got != "Alice is 30" {
		t.Errorf("Run() result = %#v, want %q", out, "Alice is 30")
	}
}

func TestEkumaRuntimeFreshVMPerRun(t *testing.T) {
	r := NewEkumaRuntime()

	prog, err := r.Compile("var x = (typeof x === 'number' ? x : 0) + 1; return x")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	for i := range 2 {
		out, err := prog.Run(nil)
		if err != nil {
			t.Fatalf("Run() #%d error = %v, want nil", i+1, err)
		}
		if got := out.(int64); got != 1 {
			t.Errorf("Run() #%d = %d, want 1 (no shared state between runs)", i+1, got)
		}
	}
}

func TestEkumaRuntimeStringInput(t *testing.T) {
	r := NewEkumaRuntime()

	prog, err := r.Compile("return input + ' world'")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run("hello")
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got, ok := out.(string); !ok || got != "hello world" {
		t.Errorf("Run() result = %#v, want %q", out, "hello world")
	}
}

func TestEkumaRuntimeNilInput(t *testing.T) {
	r := NewEkumaRuntime()

	prog, err := r.Compile("return input")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(nil)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if out != nil {
		t.Errorf("Run() with nil input = %v, want nil", out)
	}
}

func TestEkumaRuntimeObjectAndArrayResult(t *testing.T) {
	r := NewEkumaRuntime()

	prog, err := r.Compile("return { name: input.name, tags: ['a', 'b'], count: 3, ratio: 1.5 }")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{"name": "Bob"})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	want := map[string]any{
		"name":  "Bob",
		"tags":  []any{"a", "b"},
		"count": int64(3),
		"ratio": 1.5,
	}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("Run() result = %#v, want %#v", out, want)
	}
}

func TestEkumaRuntimeBooleanAndNullResult(t *testing.T) {
	r := NewEkumaRuntime()

	prog, err := r.Compile("return [true, null, undefined]")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(nil)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	want := []any{true, nil, nil}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("Run() result = %#v, want %#v", out, want)
	}
}

func TestEkumaRuntimeInterrupt(t *testing.T) {
	r := NewEkumaRuntime()

	prog, err := r.Compile("while (true) {}")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	ip, ok := prog.(interface{ Interrupt() })
	if !ok {
		t.Fatal("ekuma program does not implement Interrupt()")
	}

	time.AfterFunc(100*time.Millisecond, ip.Interrupt)

	_, err = prog.Run(nil)
	if err == nil {
		t.Fatal("Run() of infinite loop after Interrupt: expected error, got nil")
	}
	if !errors.Is(err, ErrRuntime) {
		t.Errorf("Run() error = %v, want ErrRuntime", err)
	}
}
