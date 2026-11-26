package funcstore

// FunctionInterface defines the contract for FaaS function entities.
// A function is a persisted script that can be invoked by ID or alias.
type FunctionInterface interface {
	// Identity
	// GetID returns the unique identifier of the function.
	GetID() string
	// SetID sets the unique identifier of the function.
	SetID(id string) FunctionInterface

	// Alias — human-friendly callable key (unique, optional)
	// GetAlias returns the alias of the function.
	GetAlias() string
	// SetAlias sets the alias of the function.
	SetAlias(alias string) FunctionInterface

	// Name — free-form display label (not used for lookup)
	// GetName returns the display name of the function.
	GetName() string
	// SetName sets the display name of the function.
	SetName(name string) FunctionInterface

	// Runtime — the runtime identifier (e.g. "js")
	// GetRuntime returns the runtime identifier.
	GetRuntime() string
	// SetRuntime sets the runtime identifier.
	SetRuntime(runtime string) FunctionInterface

	// Source — the script source code
	// GetSource returns the script source code.
	GetSource() string
	// SetSource sets the script source code.
	SetSource(source string) FunctionInterface

	// TimeoutMs — per-function execution timeout in milliseconds
	// GetTimeoutMs returns the timeout in milliseconds.
	GetTimeoutMs() int64
	// SetTimeoutMs sets the timeout in milliseconds.
	SetTimeoutMs(timeoutMs int64) FunctionInterface

	// MemLimit — per-function memory limit in bytes
	// GetMemLimit returns the memory limit in bytes.
	GetMemLimit() int64
	// SetMemLimit sets the memory limit in bytes.
	SetMemLimit(memLimit int64) FunctionInterface

	// Metadata — caller-defined JSON blob
	// GetMetadata returns the metadata JSON string.
	GetMetadata() string
	// SetMetadata sets the metadata JSON string.
	SetMetadata(metadata string) FunctionInterface

	// Group — the group this function belongs to, expressed as a path or ID.
	// The store resolves the value to a concrete group_id at Register time.
	// GetGroup returns the group path or ID.
	GetGroup() string
	// SetGroup sets the group path or ID.
	SetGroup(group string) FunctionInterface

	// GroupID — the resolved group ID persisted by the store.
	// GetGroupID returns the resolved group ID.
	GetGroupID() string
	// SetGroupID sets the resolved group ID.
	SetGroupID(groupID string) FunctionInterface

	// AliasPath — denormalised "group/alias" path for indexed lookup.
	// GetAliasPath returns the denormalised alias path.
	GetAliasPath() string
	// SetAliasPath sets the denormalised alias path.
	SetAliasPath(aliasPath string) FunctionInterface

	// Timestamps
	// GetCreatedAt returns the creation timestamp as a unix integer string.
	GetCreatedAt() string
	// SetCreatedAt sets the creation timestamp.
	SetCreatedAt(createdAt string) FunctionInterface
	// GetUpdatedAt returns the last update timestamp as a unix integer string.
	GetUpdatedAt() string
	// SetUpdatedAt sets the last update timestamp.
	SetUpdatedAt(updatedAt string) FunctionInterface

	// DataObject methods (from embedded dataobject.DataObject)
	// Data returns all function data as a map.
	Data() map[string]string
	// Get retrieves a value by key from the function data.
	Get(key string) string
	// Set stores a value by key in the function data.
	Set(key string, value string)
	// Hydrate populates the function with data from a map.
	Hydrate(data map[string]string)
	// IsDirty returns true if the function has unsaved changes.
	IsDirty() bool
	// MarkAsDirty marks the function as dirty.
	MarkAsDirty(columns ...string)
	// MarkAsNotDirty clears the dirty state of the function.
	MarkAsNotDirty(columns ...string)
}
