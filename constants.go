// Package funcstore provides embedded serverless function storage and execution.
//
// Store a function once, get an ID, invoke it anywhere — in any language.
// FaaS persists scripts in a database and lets you execute them by ID or alias.
// It is runtime-agnostic by design: new runtimes are pluggable, not bolted on.
//
// Backed by dracory/neat for 10+ database backends (SQLite, MySQL, Postgres, etc.).
//
// Quick start:
//
//	store, _ := funcstore.NewStore(funcstore.StoreOptions{
//	    DB:       db,
//	    Driver:   "sqlite",
//	    Runtimes: []funcstore.Runtime{funcstore.NewGojaRuntime()},
//	    AutomigrateEnabled: true,
//	})
//
//	store.FunctionCreate(ctx, funcstore.NewFunction().
//	    SetRuntime("js").
//	    SetSource("return input * 2").
//	    SetAlias("double"))
//
//	out, _ := store.FunctionInvoke(ctx, "double", 21)   // -> 42
package funcstore

import "errors"

// Version is the current package version.
const Version = "0.1.0"

// DefaultTimeoutMs is the default execution timeout if neither the function
// nor the invocation specifies one.
const DefaultTimeoutMs = 5000

// DefaultMemLimit is the default per-function memory limit in bytes.
const DefaultMemLimit = 16777216

// Typed errors returned by funcstore operations.
var (
	ErrNotFound       = errors.New("funcstore: function not found")
	ErrGroupNotFound  = errors.New("funcstore: group not found")
	ErrRuntime        = errors.New("funcstore: runtime error")
	ErrTimeout        = errors.New("funcstore: execution timed out")
	ErrPanic          = errors.New("funcstore: runtime panic")
	ErrUnknownRuntime = errors.New("funcstore: unknown runtime")
	ErrAliasTaken     = errors.New("funcstore: alias already taken")
	ErrInvalidID      = errors.New("funcstore: invalid function ID")
)

// Column names for the funcstore_functions table.
const (
	COLUMN_ID         = "id"
	COLUMN_GROUP_ID   = "group_id"
	COLUMN_ALIAS      = "alias"
	COLUMN_ALIAS_PATH = "alias_path"
	COLUMN_NAME       = "name"
	COLUMN_RUNTIME    = "runtime"
	COLUMN_SOURCE     = "source"
	COLUMN_TIMEOUT_MS = "timeout_ms"
	COLUMN_MEM_LIMIT  = "mem_limit"
	COLUMN_METADATA   = "metadata"
	COLUMN_CREATED_AT = "created_at"
	COLUMN_UPDATED_AT = "updated_at"
)

// Column names for the funcstore_groups table.
const (
	COLUMN_GROUP_ID_G        = "id"
	COLUMN_GROUP_SLUG        = "slug"
	COLUMN_GROUP_PARENT_ID   = "parent_id"
	COLUMN_GROUP_PATH        = "path"
	COLUMN_GROUP_DESCRIPTION = "description"
	COLUMN_GROUP_METADATA    = "metadata"
	COLUMN_GROUP_CREATED_AT  = "created_at"
	COLUMN_GROUP_UPDATED_AT  = "updated_at"
)

// Column names for the funcstore_function_versions table.
const (
	COLUMN_VERSION_ID          = "id"
	COLUMN_VERSION_FUNCTION_ID = "function_id"
	COLUMN_VERSION_SOURCE      = "source"
	COLUMN_VERSION_CREATED_AT  = "created_at"
)

// Runtime identifiers.
const (
	RUNTIME_JS  = "js"
	RUNTIME_GO  = "go"
	RUNTIME_PHP = "php"
)
