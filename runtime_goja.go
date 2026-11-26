package funcstore

import (
	"fmt"

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
	wrapped := "(function() { " + source + " })()"

	prog, err := goja.Compile("FaaS", wrapped, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}
	return &gojaProgram{program: prog}, nil
}

// gojaProgram implements Program. Each Run creates a fresh VM (no shared state)
// but reuses the pre-compiled bytecode.
type gojaProgram struct {
	program *goja.Program
}

func (p *gojaProgram) Run(input any) (any, error) {
	vm := goja.New()

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

// exportValue converts a goja.Value to a plain Go value.
func exportValue(v goja.Value) any {
	if v == nil {
		return nil
	}
	return v.Export()
}
