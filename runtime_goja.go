package funcstore

import (
	"fmt"
	"sync"

	"github.com/dop251/goja"
)

// gojaRuntime implements Runtime using the Goja JavaScript engine (pure Go, ES5.1+).
type gojaRuntime struct{}

// NewGojaRuntime creates a JavaScript runtime backed by Goja.
func NewGojaRuntime() Runtime {
	return &gojaRuntime{}
}

func (r *gojaRuntime) Name() string {
	return "js"
}

func (r *gojaRuntime) Compile(source string) (Program, error) {
	// Wrap the source in a function body so that `return` statements are valid.
	// This lets users write `return input * 2` instead of requiring a function wrapper.
	// The closing brace goes on its own line so a trailing "//" comment in the
	// user source cannot comment it out.
	wrapped := "(function() { " + source + "\n})()"

	prog, err := goja.Compile("FaaS", wrapped, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}
	return &gojaProgram{program: prog}, nil
}

// gojaProgram implements Program. Each Run creates a fresh VM (no shared state)
// but reuses the pre-compiled bytecode. The VM is tracked while running so
// that Interrupt can stop a runaway script from another goroutine.
type gojaProgram struct {
	program *goja.Program
	mu      sync.Mutex
	vm      *goja.Runtime
}

func (p *gojaProgram) Run(input any) (any, error) {
	vm := goja.New()

	p.mu.Lock()
	p.vm = vm
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.vm = nil
		p.mu.Unlock()
	}()

	// Bind the input as a global variable accessible to the script.
	if err := vm.Set("input", input); err != nil {
		return nil, fmt.Errorf("%w: failed to bind input: %v", ErrRuntime, err)
	}

	val, err := vm.RunProgram(p.program)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}

	return exportValue(val), nil
}

// Interrupt stops the currently executing Run, if any. It is safe to call
// from another goroutine; this is how the store enforces timeouts.
func (p *gojaProgram) Interrupt() {
	p.mu.Lock()
	vm := p.vm
	p.mu.Unlock()
	if vm != nil {
		vm.Interrupt(ErrTimeout)
	}
}

// exportValue converts a goja.Value to a plain Go value.
func exportValue(v goja.Value) any {
	if v == nil {
		return nil
	}
	return v.Export()
}
