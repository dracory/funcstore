package funcstore

import (
	"context"
	"database/sql"
)

// StoreInterface defines the persistence and execution contract for funcstore functions.
// Implementations may use any database backend; the default SQL-backed
// store (NewStore) works with any *sql.DB (SQLite, MySQL, Postgres, etc.).
type StoreInterface interface {
	// MigrateUp creates the funcstore tables and default root group.
	MigrateUp(ctx context.Context, tx ...*sql.Tx) error
	// MigrateDown drops the funcstore tables.
	MigrateDown(ctx context.Context, tx ...*sql.Tx) error

	// GroupCreate inserts a new group and returns it with resolved Path.
	// Parent may be a group ID or path; an empty Parent creates a root group.
	GroupCreate(ctx context.Context, g GroupInterface) (GroupInterface, error)
	// GroupFindByID retrieves a group by ID.
	GroupFindByID(ctx context.Context, id string) (GroupInterface, error)
	// GroupFindByPath retrieves a group by its denormalised path.
	GroupFindByPath(ctx context.Context, path string) (GroupInterface, error)
	// GroupList returns groups matching the filter.
	GroupList(ctx context.Context, filter GroupFilter) ([]GroupInterface, error)
	// GroupUpdate overwrites mutable fields of an existing group.
	GroupUpdate(ctx context.Context, g GroupInterface) error
	// GroupDelete removes a group by ID. Non-empty groups are refused unless Cascade is true.
	GroupDelete(ctx context.Context, id string, opts DeleteOpts) error

	// FunctionCreate inserts a new function record.
	// If the function's ID is empty, one is auto-generated.
	// The source is compile-checked against the registered runtime (fail-fast).
	// If the function's Group is empty, it is placed in the default root group.
	FunctionCreate(ctx context.Context, fn FunctionInterface) error

	// FunctionFindByID retrieves a function by its ID.
	FunctionFindByID(ctx context.Context, id string) (FunctionInterface, error)

	// FunctionFindByAlias retrieves a function by its alias or alias path.
	FunctionFindByAlias(ctx context.Context, alias string) (FunctionInterface, error)

	// FunctionInvoke executes a function by its ID, alias path, or alias.
	// The key is first tried as an ID; if not found, it is tried as an alias path and then an alias.
	// The input is passed to the runtime as the "input" binding.
	FunctionInvoke(ctx context.Context, key string, input any) (any, error)

	// FunctionList returns all stored functions ordered by created_at descending.
	FunctionList(ctx context.Context) ([]FunctionInterface, error)

	// FunctionUpdate overwrites all mutable fields of an existing function (matched by ID).
	FunctionUpdate(ctx context.Context, fn FunctionInterface) error

	// FunctionDelete removes a function by its ID.
	FunctionDelete(ctx context.Context, id string) error

	// FunctionVersionList returns all saved source versions for a function,
	// ordered newest-first by created_at.
	FunctionVersionList(ctx context.Context, functionID string) ([]FunctionVersionInterface, error)

	// FunctionVersionGet retrieves a single version by its ID.
	FunctionVersionGet(ctx context.Context, versionID string) (FunctionVersionInterface, error)

	// FunctionVersionDelete removes a version by its ID.
	FunctionVersionDelete(ctx context.Context, versionID string) error

	// FunctionRestore rolls back a function's source to a previous version.
	// It first snapshots the current source (so the current state is also
	// recoverable), then sets the function's source to the version's source.
	FunctionRestore(ctx context.Context, functionID string, versionID string) error

	// DB returns the underlying *sql.DB (useful for app-level integration).
	DB() *sql.DB
}

// StoreOptions configures a new SQL-backed store.
type StoreOptions struct {
	DB                 *sql.DB
	Driver             string    // "sqlite", "mysql", "postgres" — controls placeholder syntax
	TableName          string    // defaults to "funcstore_functions"
	GroupsTableName    string    // defaults to "funcstore_groups"
	VersionsTableName  string    // defaults to "funcstore_function_versions"
	AutomigrateEnabled bool      // if true, runs MigrateUp on construction
	Runtimes           []Runtime // runtimes to register (e.g. NewGojaRuntime())
}
