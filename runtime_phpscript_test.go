package funcstore

import (
	"errors"
	"os"
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

func TestPhpscriptRuntimeIncludeFromSandbox(t *testing.T) {
	t.Cleanup(func() { os.Remove("data/faas/files/include-test-lib.php") })

	r := NewPhpscriptRuntime()

	// The self-extracting multi-file pattern: the script writes a library
	// into the filesystem sandbox and then includes it by relative path.
	prog, err := r.Compile(`<?php
		if (!file_exists('include-test-lib.php')) {
			file_put_contents('include-test-lib.php', '<?php function include_test_greet() { return "hello from include"; }');
		}
		include 'include-test-lib.php';
		echo include_test_greet();
	`)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	m, _ := out.(map[string]interface{})
	body, _ := m["body"].(string)
	if body != "hello from include" {
		t.Errorf("Run() body = %q, want %q", body, "hello from include")
	}
}

func TestPhpscriptRuntimeIncludeCannotEscapeSandbox(t *testing.T) {
	r := NewPhpscriptRuntime()

	// A ".." path climbing past the sandbox root must not resolve; phpscript
	// clamps traversal at the source root.
	prog, err := r.Compile(`<?php include '../../../../etc/passwd'; echo "unreachable";`)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	_, err = prog.Run(map[string]any{})
	if err == nil {
		t.Fatal("Run() with escaping include: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "passwd") {
		t.Errorf("Run() error = %v, want it to name the missing file", err)
	}
}

// TestPhpscriptRuntimeRedirectStatus verifies the status staged by
// header('Location: ...') reaches the host as a "status" field.
func TestPhpscriptRuntimeRedirectStatus(t *testing.T) {
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
	status, ok := m["status"].(int64)
	if !ok {
		t.Fatalf("expected int64 status field, got: %v", m["status"])
	}
	if status != 302 {
		t.Errorf("expected status 302, got %d", status)
	}
}

// TestPhpscriptRuntimeExplicitStatus verifies http_response_code() carries
// through as well, and that a default 200 response adds no status field.
func TestPhpscriptRuntimeExplicitStatus(t *testing.T) {
	r := NewPhpscriptRuntime()

	prog, err := r.Compile(`<?php http_response_code(404); echo "missing";`)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}

	out, err := prog.Run(map[string]any{})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	m, _ := out.(map[string]interface{})
	if m["status"] != int64(404) {
		t.Errorf("expected status 404, got: %v", m["status"])
	}

	plain, err := r.Compile(`<?php echo "ok";`)
	if err != nil {
		t.Fatalf("Compile() error = %v, want nil", err)
	}
	out2, err := plain.Run(map[string]any{})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	m2, _ := out2.(map[string]interface{})
	if _, has := m2["status"]; has {
		t.Errorf("plain 200 response should carry no status field, got: %v", m2["status"])
	}
}
