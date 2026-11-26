package funcstore

import (
	"fmt"
	"reflect"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"golang.org/x/crypto/bcrypt"
)

// yaegiRuntime implements Runtime using Yaegi, a Go interpreter.
// Function sources are complete Go programs (package main) that must
// define: func faasRun(input map[string]interface{}) interface{}
//
// SECURITY: the interpreter exposes most of the Go standard library to
// scripts, including os, net, net/http, syscall and unsafe. Function
// sources must be treated as trusted code with the same privileges as the
// host process — this runtime is NOT a sandbox. Do not register it when
// function authors are untrusted.
type yaegiRuntime struct{}

// NewYaegiRuntime creates a Go runtime backed by Yaegi.
func NewYaegiRuntime() Runtime {
	return &yaegiRuntime{}
}

func (r *yaegiRuntime) Name() string {
	return "go"
}

// extraSymbols exposes selected non-stdlib packages to function sources.
var extraSymbols = interp.Exports{
	"golang.org/x/crypto/bcrypt/bcrypt": map[string]reflect.Value{
		"GenerateFromPassword":         reflect.ValueOf(bcrypt.GenerateFromPassword),
		"CompareHashAndPassword":       reflect.ValueOf(bcrypt.CompareHashAndPassword),
		"DefaultCost":                  reflect.ValueOf(bcrypt.DefaultCost),
		"MinCost":                      reflect.ValueOf(bcrypt.MinCost),
		"MaxCost":                      reflect.ValueOf(bcrypt.MaxCost),
		"ErrMismatchedHashAndPassword": reflect.ValueOf(&bcrypt.ErrMismatchedHashAndPassword).Elem(),
		"ErrHashTooShort":              reflect.ValueOf(&bcrypt.ErrHashTooShort).Elem(),
	},
}

func (r *yaegiRuntime) Compile(source string) (Program, error) {
	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, fmt.Errorf("%w: stdlib symbols: %v", ErrRuntime, err)
	}
	if err := i.Use(extraSymbols); err != nil {
		return nil, fmt.Errorf("%w: extra symbols: %v", ErrRuntime, err)
	}

	// The source is a complete Go program (package main, its own imports).
	// It must define: func faasRun(input map[string]interface{}) interface{}
	if _, err := i.Eval(source); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}

	v, err := i.Eval("faasRun")
	if err != nil {
		return nil, fmt.Errorf("%w: faasRun not defined: %v", ErrRuntime, err)
	}

	run, ok := v.Interface().(func(map[string]interface{}) interface{})
	if !ok {
		return nil, fmt.Errorf("%w: faasRun must have signature func(map[string]interface{}) interface{}", ErrRuntime)
	}

	return &yaegiProgram{run: run}, nil
}

// yaegiProgram implements Program. The interpreter is evaluated at Compile
// time; Run only invokes the extracted function value.
type yaegiProgram struct {
	run func(map[string]interface{}) interface{}
}

func (p *yaegiProgram) Run(input any) (any, error) {
	if input == nil {
		return p.run(map[string]interface{}{}), nil
	}
	m, ok := input.(map[string]interface{})
	if !ok {
		// Reject rather than silently substituting an empty map, so callers
		// notice type mismatches instead of debugging phantom behaviour.
		return nil, fmt.Errorf("%w: yaegi runtime requires input of type map[string]interface{}", ErrRuntime)
	}
	return p.run(m), nil
}
