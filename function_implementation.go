package funcstore

import (
	"strconv"
	"time"
	"uuid"

	"github.com/dracory/dataobject"
)

// Ensure functionImplementation implements FunctionInterface.
var _ FunctionInterface = (*functionImplementation)(nil)

// // NewFunction creates a new Function instance with default values.
// The function is initialized with a generated UUID, empty fields,
// and current timestamps.
func NewFunction() FunctionInterface {
	o := &functionImplementation{}
	o.SetID(uuid.NewV7().String()).
		SetGroup("").
		SetGroupID("").
		SetAlias("").
		SetAliasPath("").
		SetName("").
		SetRuntime("").
		SetSource("").
		SetTimeoutMs(0).
		SetMemLimit(0).
		SetMetadata("").
		SetCreatedAt(strconv.FormatInt(time.Now().Unix(), 10)).
		SetUpdatedAt(strconv.FormatInt(time.Now().Unix(), 10))
	return o
}

// functionImplementation is the concrete implementation of FunctionInterface.
// It uses dataobject.DataObject for key-value storage.
type functionImplementation struct {
	dataobject.DataObject
	groupPath string
}

// == SETTERS AND GETTERS =====================================================

func (o *functionImplementation) GetID() string {
	return o.Get(COLUMN_ID)
}

func (o *functionImplementation) SetID(id string) FunctionInterface {
	o.Set(COLUMN_ID, id)
	return o
}

func (o *functionImplementation) GetAlias() string {
	return o.Get(COLUMN_ALIAS)
}

func (o *functionImplementation) SetAlias(alias string) FunctionInterface {
	o.Set(COLUMN_ALIAS, alias)
	return o
}

func (o *functionImplementation) GetName() string {
	return o.Get(COLUMN_NAME)
}

func (o *functionImplementation) SetName(name string) FunctionInterface {
	o.Set(COLUMN_NAME, name)
	return o
}

func (o *functionImplementation) GetRuntime() string {
	return o.Get(COLUMN_RUNTIME)
}

func (o *functionImplementation) SetRuntime(runtime string) FunctionInterface {
	o.Set(COLUMN_RUNTIME, runtime)
	return o
}

func (o *functionImplementation) GetSource() string {
	return o.Get(COLUMN_SOURCE)
}

func (o *functionImplementation) SetSource(source string) FunctionInterface {
	o.Set(COLUMN_SOURCE, source)
	return o
}

func (o *functionImplementation) GetTimeoutMs() int64 {
	v := o.Get(COLUMN_TIMEOUT_MS)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (o *functionImplementation) SetTimeoutMs(timeoutMs int64) FunctionInterface {
	o.Set(COLUMN_TIMEOUT_MS, strconv.FormatInt(timeoutMs, 10))
	return o
}

func (o *functionImplementation) GetMemLimit() int64 {
	v := o.Get(COLUMN_MEM_LIMIT)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (o *functionImplementation) SetMemLimit(memLimit int64) FunctionInterface {
	o.Set(COLUMN_MEM_LIMIT, strconv.FormatInt(memLimit, 10))
	return o
}

func (o *functionImplementation) GetMetadata() string {
	return o.Get(COLUMN_METADATA)
}

func (o *functionImplementation) SetMetadata(metadata string) FunctionInterface {
	o.Set(COLUMN_METADATA, metadata)
	return o
}

func (o *functionImplementation) GetGroup() string {
	return o.groupPath
}

func (o *functionImplementation) SetGroup(group string) FunctionInterface {
	o.groupPath = group
	return o
}

func (o *functionImplementation) GetGroupID() string {
	return o.Get(COLUMN_GROUP_ID)
}

func (o *functionImplementation) SetGroupID(groupID string) FunctionInterface {
	o.Set(COLUMN_GROUP_ID, groupID)
	return o
}

func (o *functionImplementation) GetAliasPath() string {
	return o.Get(COLUMN_ALIAS_PATH)
}

func (o *functionImplementation) SetAliasPath(aliasPath string) FunctionInterface {
	o.Set(COLUMN_ALIAS_PATH, aliasPath)
	return o
}

func (o *functionImplementation) GetCreatedAt() string {
	return o.Get(COLUMN_CREATED_AT)
}

func (o *functionImplementation) SetCreatedAt(createdAt string) FunctionInterface {
	o.Set(COLUMN_CREATED_AT, createdAt)
	return o
}

func (o *functionImplementation) GetUpdatedAt() string {
	return o.Get(COLUMN_UPDATED_AT)
}

func (o *functionImplementation) SetUpdatedAt(updatedAt string) FunctionInterface {
	o.Set(COLUMN_UPDATED_AT, updatedAt)
	return o
}
