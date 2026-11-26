package funcstore

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/titpetric/phpscript/model"
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

func (r *phpscriptRuntime) Name() string {
	return "php"
}

// phpFilesRoot is the sandbox root for PHP filesystem functions. Scripts
// cannot escape it; it is shared across functions.
const phpFilesRoot = "data/faas/files"

func (r *phpscriptRuntime) Compile(source string) (Program, error) {
	var buf bytes.Buffer
	rt := runner.New(&buf, runner.Options{})
	stdlib.Register(rt)
	if err := os.MkdirAll(phpFilesRoot, 0o755); err == nil {
		stdlib.RegisterFS(rt, phpFilesRoot)
	}

	prog, err := rt.Load(source)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}

	return &phpscriptProgram{rt: rt, prog: prog, buf: &buf}, nil
}

// phpscriptProgram implements Program. The interpreter runtime is created
// at Compile time; each Run re-registers the request superglobals and
// re-executes the parsed program.
type phpscriptProgram struct {
	rt   *runner.Runtime
	prog *model.Program
	buf  *bytes.Buffer
}

func (p *phpscriptProgram) Run(input any) (any, error) {
	p.buf.Reset()

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
	c.Register(p.rt)

	if err := p.rt.Run(p.prog); err != nil {
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
		"body":    p.buf.String(),
	}, nil
}
