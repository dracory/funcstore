package funcstore

import (
	"strconv"
	"time"
	"uuid"

	"github.com/dracory/dataobject"
)

// GroupInterface defines the contract for a function group.
// A group is a named node in a self-referential tree; its Path is the
// denormalised full path ("payments/transforms").
type GroupInterface interface {
	// GetID returns the unique identifier of the group.
	GetID() string
	// SetID sets the unique identifier of the group.
	SetID(id string) GroupInterface

	// Slug — the leaf segment of the path (e.g. "transforms").
	GetSlug() string
	SetSlug(slug string) GroupInterface

	// Parent — the parent group as a path or ID. This is the input value;
	// the store resolves it to ParentID.
	GetParent() string
	SetParent(parent string) GroupInterface

	// ParentID — the resolved parent group ID persisted by the store.
	GetParentID() string
	SetParentID(parentID string) GroupInterface

	// Path — the denormalised full path (e.g. "payments/transforms").
	GetPath() string
	SetPath(path string) GroupInterface

	// Description — a human-readable summary.
	GetDescription() string
	SetDescription(description string) GroupInterface

	// Metadata — caller-defined JSON blob.
	GetMetadata() string
	SetMetadata(metadata string) GroupInterface

	// Timestamps
	GetCreatedAt() string
	SetCreatedAt(createdAt string) GroupInterface
	GetUpdatedAt() string
	SetUpdatedAt(updatedAt string) GroupInterface

	// DataObject methods
	Data() map[string]string
	Get(key string) string
	Set(key string, value string)
	Hydrate(data map[string]string)
	IsDirty() bool
	MarkAsDirty(columns ...string)
	MarkAsNotDirty(columns ...string)
}

// NewGroup creates a new Group with a generated ID and current timestamps.
func NewGroup() GroupInterface {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	g := &groupImplementation{}
	g.SetID(uuid.NewV7().String()).
		SetSlug("").
		SetParent("").
		SetParentID("").
		SetPath("").
		SetDescription("").
		SetMetadata("").
		SetCreatedAt(now).
		SetUpdatedAt(now)
	return g
}

// groupImplementation is the concrete implementation of GroupInterface.
type groupImplementation struct {
	dataobject.DataObject
	parent string
}

var _ GroupInterface = (*groupImplementation)(nil)

func (g *groupImplementation) GetID() string                  { return g.Get(COLUMN_GROUP_ID_G) }
func (g *groupImplementation) SetID(id string) GroupInterface { g.Set(COLUMN_GROUP_ID_G, id); return g }

func (g *groupImplementation) GetSlug() string { return g.Get(COLUMN_GROUP_SLUG) }
func (g *groupImplementation) SetSlug(slug string) GroupInterface {
	g.Set(COLUMN_GROUP_SLUG, slug)
	return g
}

func (g *groupImplementation) GetParent() string                      { return g.parent }
func (g *groupImplementation) SetParent(parent string) GroupInterface { g.parent = parent; return g }

func (g *groupImplementation) GetParentID() string { return g.Get(COLUMN_GROUP_PARENT_ID) }
func (g *groupImplementation) SetParentID(parentID string) GroupInterface {
	g.Set(COLUMN_GROUP_PARENT_ID, parentID)
	return g
}

func (g *groupImplementation) GetPath() string { return g.Get(COLUMN_GROUP_PATH) }
func (g *groupImplementation) SetPath(path string) GroupInterface {
	g.Set(COLUMN_GROUP_PATH, path)
	return g
}

func (g *groupImplementation) GetDescription() string { return g.Get(COLUMN_GROUP_DESCRIPTION) }
func (g *groupImplementation) SetDescription(description string) GroupInterface {
	g.Set(COLUMN_GROUP_DESCRIPTION, description)
	return g
}

func (g *groupImplementation) GetMetadata() string { return g.Get(COLUMN_GROUP_METADATA) }
func (g *groupImplementation) SetMetadata(metadata string) GroupInterface {
	g.Set(COLUMN_GROUP_METADATA, metadata)
	return g
}

func (g *groupImplementation) GetCreatedAt() string { return g.Get(COLUMN_GROUP_CREATED_AT) }
func (g *groupImplementation) SetCreatedAt(createdAt string) GroupInterface {
	g.Set(COLUMN_GROUP_CREATED_AT, createdAt)
	return g
}

func (g *groupImplementation) GetUpdatedAt() string { return g.Get(COLUMN_GROUP_UPDATED_AT) }
func (g *groupImplementation) SetUpdatedAt(updatedAt string) GroupInterface {
	g.Set(COLUMN_GROUP_UPDATED_AT, updatedAt)
	return g
}

// GroupFilter is used to list groups.
// Parent, when non-empty, matches groups whose parent path or ID equals it.
// Recursive, when true, returns all descendant groups instead of only
// direct children.
type GroupFilter struct {
	Parent    string
	Recursive bool
}

// DeleteOpts controls group deletion.
type DeleteOpts struct {
	// Cascade, when true, deletes child groups and their functions.
	// By default a non-empty group refuses to be deleted.
	Cascade bool
}
