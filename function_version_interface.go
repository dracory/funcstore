package funcstore

// FunctionVersionInterface defines the contract for a saved source snapshot
// of a function. Versions are created automatically by FunctionUpdate when
// the source code changes, and can be restored via FunctionRestore.
type FunctionVersionInterface interface {
	// GetID returns the unique identifier of the version.
	GetID() string
	// SetID sets the unique identifier of the version.
	SetID(id string) FunctionVersionInterface

	// FunctionID — the function this version belongs to.
	GetFunctionID() string
	SetFunctionID(functionID string) FunctionVersionInterface

	// Source — the function source code captured at this version.
	GetSource() string
	SetSource(source string) FunctionVersionInterface

	// CreatedAt — unix timestamp string of when the version was saved.
	GetCreatedAt() string
	SetCreatedAt(createdAt string) FunctionVersionInterface

	// DataObject methods (from embedded dataobject.DataObject)
	Data() map[string]string
	Get(key string) string
	Set(key string, value string)
	Hydrate(data map[string]string)
	IsDirty() bool
	MarkAsDirty(columns ...string)
	MarkAsNotDirty(columns ...string)
}
