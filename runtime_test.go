package funcstore

import (
	"errors"
	"testing"
)

func TestGojaRuntimeName(t *testing.T) {
	r := NewGojaRuntime()

	if got := r.Name(); got != "js" {
		t.Errorf("Name() = %q, want %q", got, "js")
	}
}

func TestGojaRuntimeCompileAndRun(t *testing.T) {
	r := NewGojaRuntime()

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

func TestGojaRuntimeCompileSyntaxError(t *testing.T) {
	r := NewGojaRuntime()

	_, err := r.Compile("var ++++ = ;;;")
	if err == nil {
		t.Fatal("Compile() with invalid syntax: expected error, got nil")
	}
	if !errors.Is(err, ErrRuntime) {
		t.Errorf("Compile() error = %v, want ErrRuntime", err)
	}
}

func TestGojaRuntimeRunThrowsError(t *testing.T) {
	r := NewGojaRuntime()

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

func TestGojaRuntimeRunWithObjectInput(t *testing.T) {
	r := NewGojaRuntime()

	prog, err := r.Compile("return input.name + ' is ' + input.age")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{"name": "Alice", "age": 30})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	got, ok := out.(string)
	if !ok {
		t.Fatalf("Run() result type = %T, want string", out)
	}
	if got != "Alice is 30" {
		t.Errorf("Run() result = %q, want %q", got, "Alice is 30")
	}
}

func TestGojaRuntimeFreshVMPerRun(t *testing.T) {
	r := NewGojaRuntime()

	// First run sets a global variable.
	prog, err := r.Compile("var x = (x || 0) + 1; return x")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out1, err := prog.Run(nil)
	if err != nil {
		t.Fatalf("Run() #1 error = %v, want nil", err)
	}
	if got := out1.(int64); got != 1 {
		t.Fatalf("Run() #1 = %d, want 1 (fresh VM)", got)
	}

	// Second run should NOT see the state from the first run.
	out2, err := prog.Run(nil)
	if err != nil {
		t.Fatalf("Run() #2 error = %v, want nil", err)
	}
	if got := out2.(int64); got != 1 {
		t.Errorf("Run() #2 = %d, want 1 (no shared state between runs)", got)
	}
}

func TestGojaRuntimeStringInput(t *testing.T) {
	r := NewGojaRuntime()

	prog, err := r.Compile("return input + ' world'")
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run("hello")
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	got, ok := out.(string)
	if !ok {
		t.Fatalf("Run() result type = %T, want string", out)
	}
	if got != "hello world" {
		t.Errorf("Run() result = %q, want %q", got, "hello world")
	}
}

func TestGojaRuntimeNilInput(t *testing.T) {
	r := NewGojaRuntime()

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
