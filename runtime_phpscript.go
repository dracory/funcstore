package funcstore

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/titpetric/phpscript/runner"
	"github.com/titpetric/phpscript/stdlib"
)

// phpscriptRuntime implements Runtime using titpetric/phpscript, a pure-Go
// PHP interpreter. Function sources are complete PHP scripts; echo/print
// output becomes the response body, and headers set via header() are
// merged into the response.
type phpscriptRuntime struct{}

// NewPhpscriptRuntime creates a PHP runtime backed by phpscript.
func NewPhpscriptRuntime() Runtime {
	return &phpscriptRuntime{}
}

// phpFilesRoot is the sandbox root for PHP filesystem functions. Scripts
// cannot escape it; it is shared across functions.
const phpFilesRoot = "data/faas/files"

// phpFilesOnce ensures the sandbox root is created at most once per process
// instead of on every compile.
var (
	phpFilesOnce sync.Once
	phpFilesOK   bool
)

func ensurePHPFilesRoot() bool {
	phpFilesOnce.Do(func() {
		phpFilesOK = os.MkdirAll(phpFilesRoot, 0o755) == nil
	})
	return phpFilesOK
}

// newRunner builds a fresh interpreter runtime writing to buf.
func newRunner(buf *bytes.Buffer) *runner.Runtime {
	opts := runner.Options{}
	if ensurePHPFilesRoot() {
		// Point the source filesystem at the same sandbox the filesystem
		// bindings use, so include/require resolve files the script itself
		// has written there. phpscript clamps ".." at the root, so scripts
		// cannot climb out of the sandbox through include paths.
		opts.RootFS = os.DirFS(phpFilesRoot)
	}
	rt := runner.New(buf, opts)
	stdlib.Register(rt)
	if ensurePHPFilesRoot() {
		stdlib.RegisterFS(rt, phpFilesRoot)
	}
	return rt
}

func (r *phpscriptRuntime) Name() string {
	return "php"
}

func (r *phpscriptRuntime) Compile(source string) (Program, error) {
	// Parse once to validate the source (fail fast); each Run re-parses into
	// its own interpreter instance so programs are safe for concurrent use
	// and share no state between invocations.
	var buf bytes.Buffer
	rt := newRunner(&buf)
	if _, err := rt.Load(source); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}

	return &phpscriptProgram{source: source}, nil
}

// phpscriptProgram implements Program. It keeps only the source text; Run
// constructs a fresh runner, output buffer, and request context per call,
// so concurrent Run calls never share mutable state.
type phpscriptProgram struct {
	source string
}

func (p *phpscriptProgram) Run(input any) (any, error) {
	var buf bytes.Buffer
	rt := newRunner(&buf)

	prog, err := rt.Load(p.source)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}

	c := runner.NewContext()
	if m, ok := input.(map[string]interface{}); ok {
		if q, ok := m["query"].(map[string]string); ok {
			c.Get = q
		}
		if h, ok := m["headers"].(map[string]string); ok {
			c.Headers = h
		}
		if b, ok := m["body"].(string); ok && b != "" {
			if vals, err := url.ParseQuery(b); err == nil {
				post := make(map[string]string, len(vals))
				for k, v := range vals {
					if len(v) > 0 {
						post[k] = v[0]
					}
				}
				c.Post = post
			}
		}
	}
	c.Register(rt)

	if err := rt.Run(prog); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}

	headers := map[string]interface{}{
		"Content-Type": "text/html; charset=utf-8",
	}
	for k, vs := range c.ResponseHeaders() {
		if len(vs) > 0 {
			headers[k] = strings.Join(vs, ", ")
		}
	}

	return map[string]interface{}{
		"headers": headers,
		"body":    buf.String(),
	}, nil
}
