package funcstore

// Program is a compiled, executable artifact produced by a Runtime.
// Implementations should be safe for concurrent Run calls.
type Program interface {
	// Run executes the compiled program with the given input and returns
	// the result. The implementation should enforce the timeout if one
	// is provided via the context by the caller (the Vault handles this).
	Run(input any) (any, error)
}

// Runtime is the pluggable interface for script execution backends.
// New runtimes (Lua, Tengo, Starlark, WASM, etc.) implement this interface
// and register with the Vault via the WithRuntime option.
type Runtime interface {
	// Name returns the runtime identifier (e.g. "js", "tengo", "lua").
	Name() string

	// Compile parses source into an executable Program.
	Compile(source string) (Program, error)
}
