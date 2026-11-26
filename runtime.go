package funcstore

// Program is a compiled, executable artifact produced by a Runtime.
// Implementations should be safe for concurrent Run calls.
//
// Implementations that support stopping a runaway execution should
// additionally implement Interrupt(). The store calls it when a function
// exceeds its timeout or the invocation context is cancelled, preventing
// abandoned goroutines from consuming CPU forever.
type Program interface {
	// Run executes the compiled program with the given input and returns
	// the result.
	Run(input any) (any, error)
}

// Runtime is the pluggable interface for script execution backends.
// New runtimes (Lua, Tengo, Starlark, WASM, etc.) implement this interface
// and register with the store via StoreOptions.Runtimes or RegisterRuntime.
type Runtime interface {
	// Name returns the runtime identifier (e.g. "js", "tengo", "lua").
	Name() string

	// Compile parses source into an executable Program.
	Compile(source string) (Program, error)
}
