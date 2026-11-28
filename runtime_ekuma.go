package funcstore

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/hajimehoshi/ekuma"
)

// ekumaRuntime implements Runtime using Ekuma, a pure Go ES2026 JavaScript engine.
// Ekuma's Go API is unstable; the module is pinned in go.mod.
type ekumaRuntime struct{}

// NewEkumaRuntime creates a JavaScript runtime backed by Ekuma.
func NewEkumaRuntime() Runtime {
	return &ekumaRuntime{}
}

func (r *ekumaRuntime) Name() string {
	return "js-ekuma"
}

func (r *ekumaRuntime) Compile(source string) (Program, error) {
	// Same wrapping as the goja runtime so `return` is valid in user sources.
	wrapped := "(function() { " + source + "\n})()"

	script, err := ekuma.ParseUnboundScript("FaaS", wrapped)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}
	return &ekumaProgram{script: script}, nil
}

// ekumaProgram implements Program. Each Run creates a fresh VM (no shared state)
// but reuses the pre-parsed script. The running evaluation is tracked so that
// Interrupt can stop a runaway script from another goroutine.
type ekumaProgram struct {
	script *ekuma.UnboundScript
	mu     sync.Mutex
	cancel context.CancelCauseFunc
}

func (p *ekumaProgram) Run(input any) (result any, err error) {
	// Ekuma panics on Go values it cannot convert; report them as runtime errors.
	defer func() {
		if r := recover(); r != nil {
			result, err = nil, fmt.Errorf("%w: %v", ErrRuntime, r)
		}
	}()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	p.mu.Lock()
	p.cancel = cancel
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.cancel = nil
		p.mu.Unlock()
	}()

	var vm ekuma.VM
	if err := vm.Global().Set(ctx, "input", vm.ValueOf(input)); err != nil {
		return nil, fmt.Errorf("%w: failed to bind input: %v", ErrRuntime, err)
	}

	val, err := vm.BindScript(p.script).Evaluate(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}

	out, err := exportEkumaValue(ctx, &vm, val)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntime, err)
	}
	return out, nil
}

// Interrupt stops the currently executing Run, if any. It is safe to call
// from another goroutine; this is how the store enforces timeouts.
func (p *ekumaProgram) Interrupt() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		p.cancel(ErrTimeout)
	}
}

// exportEkumaValue converts an Ekuma value to a plain Go value. Integral numbers
// become int64 and other numbers float64, matching the goja runtime. Objects and
// arrays are converted through JSON, since Ekuma does not expose object keys.
func exportEkumaValue(ctx context.Context, vm *ekuma.VM, v ekuma.Value) (any, error) {
	switch v.Type() {
	case ekuma.TypeUndefined, ekuma.TypeNull:
		return nil, nil
	case ekuma.TypeBoolean:
		return v.Bool(), nil
	case ekuma.TypeNumber:
		f := v.Float()
		if f == math.Trunc(f) && math.Abs(f) < 1<<63 {
			return int64(f), nil
		}
		return f, nil
	case ekuma.TypeString:
		return v.String(), nil
	case ekuma.TypeBigInt:
		return v.BigInt(), nil
	case ekuma.TypeObject:
		return exportEkumaJSON(ctx, vm, v)
	default:
		return nil, fmt.Errorf("unsupported result type %s", v.Type())
	}
}

func exportEkumaJSON(ctx context.Context, vm *ekuma.VM, v ekuma.Value) (any, error) {
	jsonObj, err := vm.Global().Get(ctx, "JSON")
	if err != nil {
		return nil, err
	}
	s, err := jsonObj.Call(ctx, "stringify", v)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(strings.NewReader(s.String()))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return normalizeJSONNumbers(out), nil
}

// normalizeJSONNumbers replaces json.Number with int64 or float64, matching
// exportEkumaValue for numbers.
func normalizeJSONNumbers(x any) any {
	switch t := x.(type) {
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		f, _ := t.Float64()
		return f
	case map[string]any:
		for k, e := range t {
			t[k] = normalizeJSONNumbers(e)
		}
	case []any:
		for i, e := range t {
			t[i] = normalizeJSONNumbers(e)
		}
	}
	return x
}
