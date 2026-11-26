package funcstore

import (
	"strconv"
	"time"
	"uuid"

	"github.com/dracory/dataobject"
)

// Ensure functionVersionImplementation implements FunctionVersionInterface.
var _ FunctionVersionInterface = (*functionVersionImplementation)(nil)

// NewFunctionVersion creates a new FunctionVersion with a generated ID and
// current timestamp.
func NewFunctionVersion() FunctionVersionInterface {
	o := &functionVersionImplementation{}
	o.SetID(uuid.NewV7().String()).
		SetFunctionID("").
		SetSource("").
		SetCreatedAt(strconv.FormatInt(time.Now().Unix(), 10))
	return o
}

// functionVersionImplementation is the concrete implementation of
// FunctionVersionInterface. It uses dataobject.DataObject for key-value
// storage.
type functionVersionImplementation struct {
	dataobject.DataObject
}

// == SETTERS AND GETTERS =====================================================

func (o *functionVersionImplementation) GetID() string {
	return o.Get(COLUMN_VERSION_ID)
}

func (o *functionVersionImplementation) SetID(id string) FunctionVersionInterface {
	o.Set(COLUMN_VERSION_ID, id)
	return o
}

func (o *functionVersionImplementation) GetFunctionID() string {
	return o.Get(COLUMN_VERSION_FUNCTION_ID)
}

func (o *functionVersionImplementation) SetFunctionID(functionID string) FunctionVersionInterface {
	o.Set(COLUMN_VERSION_FUNCTION_ID, functionID)
	return o
}

func (o *functionVersionImplementation) GetSource() string {
	return o.Get(COLUMN_VERSION_SOURCE)
}

func (o *functionVersionImplementation) SetSource(source string) FunctionVersionInterface {
	o.Set(COLUMN_VERSION_SOURCE, source)
	return o
}

func (o *functionVersionImplementation) GetCreatedAt() string {
	return o.Get(COLUMN_VERSION_CREATED_AT)
}

func (o *functionVersionImplementation) SetCreatedAt(createdAt string) FunctionVersionInterface {
	o.Set(COLUMN_VERSION_CREATED_AT, createdAt)
	return o
}
