package funcstore

import (
	"errors"
	"testing"
)

func TestYaegiRuntimeName(t *testing.T) {
	r := NewYaegiRuntime()

	if got := r.Name(); got != "go" {
		t.Errorf("Name() = %q, want %q", got, "go")
	}
}

func TestYaegiRuntimeCompileAndRun(t *testing.T) {
	r := NewYaegiRuntime()

	source := `package main

func faasRun(input map[string]interface{}) interface{} {
	name, _ := input["name"].(string)
	return "hello " + name
}
`
	prog, err := r.Compile(source)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{"name": "world"})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if got, ok := out.(string); !ok || got != "hello world" {
		t.Errorf("Run() result = %v, want %q", out, "hello world")
	}
}

func TestYaegiRuntimeCompileSyntaxError(t *testing.T) {
	r := NewYaegiRuntime()

	_, err := r.Compile("package main\n\nfunc faasRun( {{{")
	if err == nil {
		t.Fatal("Compile() with invalid syntax: expected error, got nil")
	}
	if !errors.Is(err, ErrRuntime) {
		t.Errorf("Compile() error = %v, want ErrRuntime", err)
	}
}

func TestYaegiRuntimeMissingEntrypoint(t *testing.T) {
	r := NewYaegiRuntime()

	_, err := r.Compile("package main\n\nfunc other() {}")
	if err == nil {
		t.Fatal("Compile() without faasRun: expected error, got nil")
	}
	if !errors.Is(err, ErrRuntime) {
		t.Errorf("Compile() error = %v, want ErrRuntime", err)
	}
}

func TestYaegiRuntimeStdlibAndBcrypt(t *testing.T) {
	r := NewYaegiRuntime()

	source := `package main

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func faasRun(input map[string]interface{}) interface{} {
	password, _ := input["password"].(string)
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err.Error()
	}
	return strings.HasPrefix(string(h), "$2")
}
`
	prog, err := r.Compile(source)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{"password": "secret123"})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if got, ok := out.(bool); !ok || !got {
		t.Errorf("Run() result = %v, want true (bcrypt $2 prefix)", out)
	}
}

// TestYaegiRuntimeNilInput verifies a nil input is converted to an empty
// map before reaching the function.
func TestYaegiRuntimeNilInput(t *testing.T) {
	r := NewYaegiRuntime()

	source := `package main

func faasRun(input map[string]interface{}) interface{} {
	if len(input) == 0 {
		return "empty"
	}
	return "nonempty"
}
`
	prog, err := r.Compile(source)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(nil)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got, ok := out.(string); !ok || got != "empty" {
		t.Errorf("Run() with nil input = %v, want %q (empty map)", out, "empty")
	}
}
