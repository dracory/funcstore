package funcstore

import (
	"errors"
	"strings"
	"testing"
)

func TestPhpscriptRuntimeName(t *testing.T) {
	r := NewPhpscriptRuntime()

	if got := r.Name(); got != "php" {
		t.Errorf("Name() = %q, want %q", got, "php")
	}
}

func TestPhpscriptRuntimeCompileAndRun(t *testing.T) {
	r := NewPhpscriptRuntime()

	prog, err := r.Compile(`<?php echo "Hello, " . $_GET["name"] . "!";`)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{
		"query": map[string]string{"name": "Devin"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	m, ok := out.(map[string]interface{})
	if !ok {
		t.Fatalf("Run() result type = %T, want map", out)
	}
	body, _ := m["body"].(string)
	if body != "Hello, Devin!" {
		t.Errorf("Run() body = %q, want %q", body, "Hello, Devin!")
	}
}

func TestPhpscriptRuntimePostBody(t *testing.T) {
	r := NewPhpscriptRuntime()

	prog, err := r.Compile(`<?php echo $_POST["field"];`)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{
		"body": "field=hello+world",
	})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	m, _ := out.(map[string]interface{})
	body, _ := m["body"].(string)
	if body != "hello world" {
		t.Errorf("Run() body = %q, want %q", body, "hello world")
	}
}

func TestPhpscriptRuntimeCompileSyntaxError(t *testing.T) {
	r := NewPhpscriptRuntime()

	_, err := r.Compile("<?php echo ;;; {{{")
	if err == nil {
		t.Fatal("Compile() with invalid syntax: expected error, got nil")
	}
	if !errors.Is(err, ErrRuntime) {
		t.Errorf("Compile() error = %v, want ErrRuntime", err)
	}
}

func TestPhpscriptRuntimeHeaderPropagation(t *testing.T) {
	r := NewPhpscriptRuntime()

	prog, err := r.Compile(`<?php header("Location: /target"); echo "redirecting";`)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	m, _ := out.(map[string]interface{})
	headers, _ := m["headers"].(map[string]interface{})
	loc, _ := headers["Location"].(string)
	if !strings.Contains(loc, "/target") {
		t.Errorf("expected Location header containing /target, got: %v", headers)
	}
}
