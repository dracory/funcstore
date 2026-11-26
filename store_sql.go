package funcstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/dracory/neat"
	"github.com/dracory/neat/contracts/database/schema"
)

// storeSQL is the default SQL-backed StoreInterface implementation.
// It works with any *sql.DB — SQLite, MySQL, Postgres, SQL Server, etc.
// The *sql.DB is wrapped in a neat.Database for schema management.
type storeSQL struct {
	db                *sql.DB
	neatDB            *neat.Database
	driver            string
	tableName         string
	groupsTableName   string
	versionsTableName string
	runtimes          map[string]Runtime
	mu                sync.RWMutex
}

// Ensure storeSQL implements StoreInterface.
var _ StoreInterface = (*storeSQL)(nil)

// NewStore creates a SQL-backed StoreInterface from the given options.
func NewStore(opts StoreOptions) (StoreInterface, error) {
	if opts.DB == nil {
		return nil, errors.New("funcstore: StoreOptions.DB is required")
	}

	driver := strings.ToLower(strings.TrimSpace(opts.Driver))
	tableName := opts.TableName
	if tableName == "" {
		tableName = "funcstore_functions"
	}
	groupsTableName := opts.GroupsTableName
	if groupsTableName == "" {
		groupsTableName = "funcstore_groups"
	}
	versionsTableName := opts.VersionsTableName
	if versionsTableName == "" {
		versionsTableName = "funcstore_function_versions"
	}

	neatDB, err := neat.NewFromSQLDB(opts.DB)
	if err != nil {
		return nil, fmt.Errorf("funcstore: failed to create neat database: %w", err)
	}

	runtimes := make(map[string]Runtime)
	for _, r := range opts.Runtimes {
		runtimes[r.Name()] = r
	}

	s := &storeSQL{
		db:                opts.DB,
		neatDB:            neatDB,
		driver:            driver,
		tableName:         tableName,
		groupsTableName:   groupsTableName,
		versionsTableName: versionsTableName,
		runtimes:          runtimes,
	}

	if opts.AutomigrateEnabled {
		if err := s.MigrateUp(context.Background()); err != nil {
			return nil, fmt.Errorf("funcstore: auto-migrate failed: %w", err)
		}
	}

	return s, nil
}

// RegisterRuntime adds a Runtime to the store.
func (s *storeSQL) RegisterRuntime(r Runtime) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimes[r.Name()] = r
}

// MigrateUp creates the groups and functions tables if they do not exist
// and ensures a "default" root group exists.
func (s *storeSQL) MigrateUp(ctx context.Context) error {
	if !s.neatDB.Schema().HasTable(s.groupsTableName) {
		if err := s.neatDB.Schema().Create(s.groupsTableName, func(table schema.Blueprint) {
			table.String(COLUMN_GROUP_ID_G, 36)
			table.Primary(COLUMN_GROUP_ID_G)
			table.String(COLUMN_GROUP_SLUG, 255)
			table.String(COLUMN_GROUP_PARENT_ID, 36).Nullable()
			table.String(COLUMN_GROUP_PATH, 255)
			table.Unique(COLUMN_GROUP_PATH)
			table.Unique(COLUMN_GROUP_PARENT_ID, COLUMN_GROUP_SLUG)
			table.String(COLUMN_GROUP_DESCRIPTION, 255).Nullable()
			table.Text(COLUMN_GROUP_METADATA).Nullable()
			table.Integer(COLUMN_GROUP_CREATED_AT)
			table.Integer(COLUMN_GROUP_UPDATED_AT)
		}); err != nil {
			return fmt.Errorf("funcstore: create groups table: %w", err)
		}
	}

	if !s.neatDB.Schema().HasTable(s.tableName) {
		if err := s.neatDB.Schema().Create(s.tableName, func(table schema.Blueprint) {
			table.String(COLUMN_ID, 36)
			table.Primary(COLUMN_ID)
			table.String(COLUMN_GROUP_ID, 36)
			table.Index(COLUMN_GROUP_ID)
			table.String(COLUMN_ALIAS, 255).Nullable()
			table.String(COLUMN_ALIAS_PATH, 512).Nullable()
			table.Unique(COLUMN_ALIAS_PATH)
			table.String(COLUMN_NAME, 255).Nullable()
			table.String(COLUMN_RUNTIME, 50)
			table.Index(COLUMN_RUNTIME)
			table.Text(COLUMN_SOURCE)
			table.Integer(COLUMN_TIMEOUT_MS).Default(0)
			table.Integer(COLUMN_MEM_LIMIT).Default(0)
			table.Text(COLUMN_METADATA).Nullable()
			table.Integer(COLUMN_CREATED_AT)
			table.Integer(COLUMN_UPDATED_AT)
			table.Unique(COLUMN_GROUP_ID, COLUMN_ALIAS)
		}); err != nil {
			return fmt.Errorf("funcstore: create functions table: %w", err)
		}
	}

	if !s.neatDB.Schema().HasTable(s.versionsTableName) {
		if err := s.neatDB.Schema().Create(s.versionsTableName, func(table schema.Blueprint) {
			table.String(COLUMN_VERSION_ID, 36)
			table.Primary(COLUMN_VERSION_ID)
			table.String(COLUMN_VERSION_FUNCTION_ID, 36)
			table.Index(COLUMN_VERSION_FUNCTION_ID)
			table.Text(COLUMN_VERSION_SOURCE)
			table.Integer(COLUMN_VERSION_CREATED_AT)
		}); err != nil {
			return fmt.Errorf("funcstore: create function versions table: %w", err)
		}
	}

	_, err := s.ensureDefaultGroup(ctx)
	if err != nil {
		return fmt.Errorf("funcstore: ensure default group: %w", err)
	}

	return nil
}

// MigrateDown drops the functions, versions, and groups tables if they exist.
func (s *storeSQL) MigrateDown(ctx context.Context) error {
	if err := s.neatDB.Schema().DropIfExists(s.versionsTableName); err != nil {
		return err
	}
	if err := s.neatDB.Schema().DropIfExists(s.tableName); err != nil {
		return err
	}
	return s.neatDB.Schema().DropIfExists(s.groupsTableName)
}

// === Groups ================================================================

func (s *storeSQL) GroupCreate(ctx context.Context, g GroupInterface) (GroupInterface, error) {
	if g == nil {
		return nil, errors.New("funcstore: group is nil")
	}
	if strings.TrimSpace(g.GetSlug()) == "" {
		return nil, errors.New("funcstore: group slug is required")
	}
	if strings.Contains(g.GetSlug(), "/") {
		return nil, errors.New("funcstore: group slug cannot contain '/'")
	}

	parentID := ""
	parentPath := ""
	if parentKey := strings.TrimSpace(g.GetParent()); parentKey != "" {
		parent, err := s.findGroup(ctx, parentKey)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrGroupNotFound
			}
			return nil, err
		}
		parentID = parent.GetID()
		parentPath = parent.GetPath()
	}

	now := strconv.FormatInt(time.Now().Unix(), 10)
	if g.GetID() == "" {
		g.SetID(uuid.NewV7().String())
	}
	if g.GetCreatedAt() == "" {
		g.SetCreatedAt(now)
	}
	g.SetUpdatedAt(now)
	g.SetParentID(parentID)
	if parentPath == "" {
		g.SetPath(g.GetSlug())
	} else {
		g.SetPath(parentPath + "/" + g.GetSlug())
	}

	q := fmt.Sprintf(`INSERT INTO %s (id, slug, parent_id, path, description, metadata, created_at, updated_at) VALUES (%s)`,
		s.groupsTableName, s.placeholders(8))

	_, err := s.db.ExecContext(ctx, q,
		g.GetID(), g.GetSlug(), nullIfEmpty(g.GetParentID()), g.GetPath(),
		nullIfEmpty(g.GetDescription()), nullIfEmpty(g.GetMetadata()), g.GetCreatedAt(), g.GetUpdatedAt(),
	)
	if err != nil {
		return nil, fmt.Errorf("funcstore: create group: %w", err)
	}
	return g, nil
}

func (s *storeSQL) GroupFindByID(ctx context.Context, id string) (GroupInterface, error) {
	if id == "" {
		return nil, ErrNotFound
	}
	q := fmt.Sprintf("SELECT id, slug, parent_id, path, description, metadata, created_at, updated_at FROM %s WHERE id = %s",
		s.groupsTableName, s.placeholder(1))
	row := s.db.QueryRowContext(ctx, q, id)
	return s.scanGroup(row)
}

func (s *storeSQL) GroupFindByPath(ctx context.Context, path string) (GroupInterface, error) {
	if path == "" {
		return nil, ErrNotFound
	}
	q := fmt.Sprintf("SELECT id, slug, parent_id, path, description, metadata, created_at, updated_at FROM %s WHERE path = %s",
		s.groupsTableName, s.placeholder(1))
	row := s.db.QueryRowContext(ctx, q, path)
	return s.scanGroup(row)
}

func (s *storeSQL) GroupList(ctx context.Context, filter GroupFilter) ([]GroupInterface, error) {
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(filter.Parent) != "" {
		parent, err := s.findGroup(ctx, filter.Parent)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return []GroupInterface{}, nil
			}
			return nil, err
		}
		if filter.Recursive {
			pattern := escapeLike(parent.GetPath()) + "/%"
			q := fmt.Sprintf("SELECT id, slug, parent_id, path, description, metadata, created_at, updated_at FROM %s WHERE path LIKE %s ESCAPE '\\' ORDER BY path",
				s.groupsTableName, s.placeholder(1))
			rows, err = s.db.QueryContext(ctx, q, pattern)
		} else {
			q := fmt.Sprintf("SELECT id, slug, parent_id, path, description, metadata, created_at, updated_at FROM %s WHERE parent_id = %s ORDER BY slug",
				s.groupsTableName, s.placeholder(1))
			rows, err = s.db.QueryContext(ctx, q, parent.GetID())
		}
	} else {
		q := fmt.Sprintf("SELECT id, slug, parent_id, path, description, metadata, created_at, updated_at FROM %s ORDER BY path",
			s.groupsTableName)
		rows, err = s.db.QueryContext(ctx, q)
	}
	if err != nil {
		return nil, fmt.Errorf("funcstore: list groups: %w", err)
	}
	defer rows.Close()

	var groups []GroupInterface
	for rows.Next() {
		g, err := s.scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("funcstore: scan group: %w", err)
		}
		groups = append(groups, g)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("funcstore: list groups rows: %w", err)
	}
	return groups, nil
}

func (s *storeSQL) GroupUpdate(ctx context.Context, g GroupInterface) error {
	if g == nil || g.GetID() == "" {
		return ErrInvalidID
	}
	g.SetUpdatedAt(strconv.FormatInt(time.Now().Unix(), 10))
	q := fmt.Sprintf(`UPDATE %s SET description = %s, metadata = %s, updated_at = %s WHERE id = %s`,
		s.groupsTableName,
		s.placeholder(1), s.placeholder(2), s.placeholder(3), s.placeholder(4),
	)
	res, err := s.db.ExecContext(ctx, q,
		nullIfEmpty(g.GetDescription()), nullIfEmpty(g.GetMetadata()), g.GetUpdatedAt(), g.GetID(),
	)
	if err != nil {
		return fmt.Errorf("funcstore: update group: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *storeSQL) GroupDelete(ctx context.Context, id string, opts DeleteOpts) error {
	if id == "" {
		return ErrInvalidID
	}

	g, err := s.GroupFindByID(ctx, id)
	if err != nil {
		return err
	}

	children, err := s.GroupList(ctx, GroupFilter{Parent: g.GetID()})
	if err != nil {
		return err
	}
	if len(children) > 0 && !opts.Cascade {
		return errors.New("funcstore: group is not empty")
	}

	var functionCount int
	qCount := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE group_id = %s", s.tableName, s.placeholder(1))
	if err := s.db.QueryRowContext(ctx, qCount, g.GetID()).Scan(&functionCount); err != nil {
		return fmt.Errorf("funcstore: count group functions: %w", err)
	}
	if functionCount > 0 && !opts.Cascade {
		return errors.New("funcstore: group is not empty")
	}

	if opts.Cascade {
		// Delete versions, functions, and groups in this group and its
		// descendants — atomically, so a failure partway through cannot
		// leave the store half-deleted.
		pattern := escapeLike(g.GetPath()) + "/%"
		groupIDs := fmt.Sprintf("(SELECT id FROM %s WHERE path LIKE %s ESCAPE '\\' OR id = %s)",
			s.groupsTableName, s.placeholder(1), s.placeholder(2))

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("funcstore: begin delete tx: %w", err)
		}
		defer tx.Rollback()

		qDelVersions := fmt.Sprintf("DELETE FROM %s WHERE function_id IN (SELECT id FROM %s WHERE group_id IN %s)",
			s.versionsTableName, s.tableName, groupIDs)
		if _, err := tx.ExecContext(ctx, qDelVersions, pattern, g.GetID()); err != nil {
			return fmt.Errorf("funcstore: delete group function versions: %w", err)
		}

		qDelFuncs := fmt.Sprintf("DELETE FROM %s WHERE group_id IN %s", s.tableName, groupIDs)
		if _, err := tx.ExecContext(ctx, qDelFuncs, pattern, g.GetID()); err != nil {
			return fmt.Errorf("funcstore: delete group functions: %w", err)
		}

		qDelGroups := fmt.Sprintf("DELETE FROM %s WHERE path LIKE %s ESCAPE '\\' OR id = %s",
			s.groupsTableName, s.placeholder(1), s.placeholder(2))
		if _, err := tx.ExecContext(ctx, qDelGroups, pattern, g.GetID()); err != nil {
			return fmt.Errorf("funcstore: delete groups: %w", err)
		}
		return tx.Commit()
	}

	// Delete group only.
	q := fmt.Sprintf("DELETE FROM %s WHERE id = %s", s.groupsTableName, s.placeholder(1))
	res, err := s.db.ExecContext(ctx, q, g.GetID())
	if err != nil {
		return fmt.Errorf("funcstore: delete group: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// === Functions =============================================================

func (s *storeSQL) FunctionCreate(ctx context.Context, fn FunctionInterface) error {
	if fn.GetRuntime() == "" {
		return errors.New("funcstore: Runtime is required")
	}
	if fn.GetSource() == "" {
		return errors.New("funcstore: Source is required")
	}

	// Verify the runtime is registered and source compiles (fail fast).
	r := s.getRuntime(fn.GetRuntime())
	if r == nil {
		return fmt.Errorf("%w: %q", ErrUnknownRuntime, fn.GetRuntime())
	}
	if _, err := r.Compile(fn.GetSource()); err != nil {
		return err
	}

	group, err := s.resolveGroupForFunction(ctx, fn)
	if err != nil {
		return err
	}

	fn.SetGroupID(group.GetID()).SetGroup(group.GetPath())
	if err := s.validateAndSetAliasPath(ctx, fn, ""); err != nil {
		return err
	}

	if fn.GetID() == "" {
		fn.SetID(uuid.NewV7().String())
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	if fn.GetCreatedAt() == "" {
		fn.SetCreatedAt(now)
	}
	fn.SetUpdatedAt(now)

	if fn.GetTimeoutMs() <= 0 {
		fn.SetTimeoutMs(DefaultTimeoutMs)
	}
	if fn.GetMemLimit() <= 0 {
		fn.SetMemLimit(DefaultMemLimit)
	}

	q := fmt.Sprintf(`INSERT INTO %s (id, group_id, alias, alias_path, name, runtime, source, timeout_ms, mem_limit, metadata, created_at, updated_at)
		VALUES (%s)`, s.tableName, s.placeholders(12))

	_, err = s.db.ExecContext(ctx, q,
		fn.GetID(), fn.GetGroupID(), nullIfEmpty(fn.GetAlias()), nullIfEmpty(fn.GetAliasPath()),
		nullIfEmpty(fn.GetName()), fn.GetRuntime(), fn.GetSource(), fn.GetTimeoutMs(), fn.GetMemLimit(),
		nullIfEmpty(fn.GetMetadata()), fn.GetCreatedAt(), fn.GetUpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("funcstore: insert: %w", err)
	}

	return nil
}

func (s *storeSQL) FunctionFindByID(ctx context.Context, id string) (FunctionInterface, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	q := fmt.Sprintf("SELECT id, group_id, alias, alias_path, name, runtime, source, timeout_ms, mem_limit, metadata, created_at, updated_at FROM %s WHERE id = %s",
		s.tableName, s.placeholder(1))

	row := s.db.QueryRowContext(ctx, q, id)
	fn, err := s.scanFunction(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("funcstore: find by id: %w", err)
	}
	return fn, nil
}

func (s *storeSQL) FunctionFindByAlias(ctx context.Context, alias string) (FunctionInterface, error) {
	if alias == "" {
		return nil, ErrNotFound
	}

	// If the key contains a slash, treat it as a full alias path.
	if strings.Contains(alias, "/") {
		q := fmt.Sprintf("SELECT id, group_id, alias, alias_path, name, runtime, source, timeout_ms, mem_limit, metadata, created_at, updated_at FROM %s WHERE alias_path = %s",
			s.tableName, s.placeholder(1))
		row := s.db.QueryRowContext(ctx, q, alias)
		return s.scanFunction(row)
	}

	// Otherwise treat it as an alias leaf (global fallback, mainly for the default group).
	// The same leaf alias may exist in several groups; rather than returning an
	// arbitrary one, report the ambiguity so the caller can use the full path.
	q := fmt.Sprintf("SELECT id, group_id, alias, alias_path, name, runtime, source, timeout_ms, mem_limit, metadata, created_at, updated_at FROM %s WHERE alias = %s",
		s.tableName, s.placeholder(1))
	rows, err := s.db.QueryContext(ctx, q, alias)
	if err != nil {
		return nil, fmt.Errorf("funcstore: find by alias: %w", err)
	}
	defer rows.Close()

	var found FunctionInterface
	count := 0
	for rows.Next() {
		count++
		if count == 1 {
			found, err = s.scanFunction(rows)
			if err != nil {
				return nil, fmt.Errorf("funcstore: find by alias scan: %w", err)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("funcstore: find by alias rows: %w", err)
	}
	if count == 0 {
		return nil, ErrNotFound
	}
	if count > 1 {
		return nil, fmt.Errorf("%w: %q (use the full alias path)", ErrAliasAmbiguous, alias)
	}
	return found, nil
}

func (s *storeSQL) FunctionInvoke(ctx context.Context, key string, input any) (any, error) {
	if key == "" {
		return nil, ErrInvalidID
	}

	fn, err := s.resolve(ctx, key)
	if err != nil {
		return nil, err
	}

	r := s.getRuntime(fn.GetRuntime())
	if r == nil {
		return nil, fmt.Errorf("%w: %q", ErrUnknownRuntime, fn.GetRuntime())
	}

	prog, err := r.Compile(fn.GetSource())
	if err != nil {
		return nil, err
	}

	timeoutMs := fn.GetTimeoutMs()
	if timeoutMs <= 0 {
		timeoutMs = DefaultTimeoutMs
	}

	return s.runWithTimeout(ctx, prog, input, time.Duration(timeoutMs)*time.Millisecond)
}

func (s *storeSQL) FunctionList(ctx context.Context) ([]FunctionInterface, error) {
	q := fmt.Sprintf("SELECT id, group_id, alias, alias_path, name, runtime, source, timeout_ms, mem_limit, metadata, created_at, updated_at FROM %s ORDER BY created_at DESC",
		s.tableName)

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("funcstore: list: %w", err)
	}
	defer rows.Close()

	var fns []FunctionInterface
	for rows.Next() {
		fn, err := s.scanFunction(rows)
		if err != nil {
			return nil, fmt.Errorf("funcstore: list scan: %w", err)
		}
		fns = append(fns, fn)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("funcstore: list rows: %w", err)
	}
	return fns, nil
}

func (s *storeSQL) FunctionUpdate(ctx context.Context, fn FunctionInterface) error {
	if fn.GetID() == "" {
		return ErrInvalidID
	}

	// Verify the runtime is registered and the new source compiles, matching
	// the fail-fast behaviour of FunctionCreate. This runs before any
	// snapshot so a rejected update leaves no trace.
	if fn.GetRuntime() == "" {
		return errors.New("funcstore: Runtime is required")
	}
	r := s.getRuntime(fn.GetRuntime())
	if r == nil {
		return fmt.Errorf("%w: %q", ErrUnknownRuntime, fn.GetRuntime())
	}
	if _, err := r.Compile(fn.GetSource()); err != nil {
		return err
	}

	// Auto-snapshot: if the source is changing, save the old source to the
	// versions table before overwriting it.
	existing, err := s.FunctionFindByID(ctx, fn.GetID())
	if err != nil {
		return err
	}
	if existing.GetSource() != fn.GetSource() {
		if err := s.snapshotVersion(ctx, fn.GetID(), existing.GetSource()); err != nil {
			return fmt.Errorf("funcstore: snapshot version: %w", err)
		}
	}

	groupID := fn.GetGroupID()
	if strings.TrimSpace(fn.GetGroup()) != "" {
		g, err := s.resolveGroupForFunction(ctx, fn)
		if err != nil {
			return err
		}
		groupID = g.GetID()
	}
	if groupID == "" {
		g, err := s.ensureDefaultGroup(ctx)
		if err != nil {
			return err
		}
		groupID = g.GetID()
	}
	fn.SetGroupID(groupID)

	if err := s.validateAndSetAliasPath(ctx, fn, fn.GetID()); err != nil {
		return err
	}

	fn.SetUpdatedAt(strconv.FormatInt(time.Now().Unix(), 10))

	q := fmt.Sprintf(`UPDATE %s SET group_id = %s, alias = %s, alias_path = %s, name = %s, runtime = %s, source = %s,
		timeout_ms = %s, mem_limit = %s, metadata = %s, updated_at = %s WHERE id = %s`,
		s.tableName,
		s.placeholder(1), s.placeholder(2), s.placeholder(3), s.placeholder(4),
		s.placeholder(5), s.placeholder(6), s.placeholder(7), s.placeholder(8),
		s.placeholder(9), s.placeholder(10), s.placeholder(11),
	)

	res, err := s.db.ExecContext(ctx, q,
		fn.GetGroupID(), nullIfEmpty(fn.GetAlias()), nullIfEmpty(fn.GetAliasPath()), nullIfEmpty(fn.GetName()),
		fn.GetRuntime(), fn.GetSource(), fn.GetTimeoutMs(), fn.GetMemLimit(), nullIfEmpty(fn.GetMetadata()),
		fn.GetUpdatedAt(), fn.GetID(),
	)
	if err != nil {
		return fmt.Errorf("funcstore: update: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *storeSQL) FunctionDelete(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	// Delete the function and its saved versions atomically so no orphaned
	// version rows are left behind.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("funcstore: begin delete tx: %w", err)
	}
	defer tx.Rollback()

	qv := fmt.Sprintf("DELETE FROM %s WHERE function_id = %s", s.versionsTableName, s.placeholder(1))
	if _, err := tx.ExecContext(ctx, qv, id); err != nil {
		return fmt.Errorf("funcstore: delete function versions: %w", err)
	}

	q := fmt.Sprintf("DELETE FROM %s WHERE id = %s", s.tableName, s.placeholder(1))
	res, err := tx.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("funcstore: delete: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// === Function Versions =====================================================

func (s *storeSQL) FunctionVersionList(ctx context.Context, functionID string) ([]FunctionVersionInterface, error) {
	if functionID == "" {
		return nil, ErrInvalidID
	}

	q := fmt.Sprintf("SELECT id, function_id, source, created_at FROM %s WHERE function_id = %s ORDER BY id DESC",
		s.versionsTableName, s.placeholder(1))

	rows, err := s.db.QueryContext(ctx, q, functionID)
	if err != nil {
		return nil, fmt.Errorf("funcstore: version list: %w", err)
	}
	defer rows.Close()

	var versions []FunctionVersionInterface
	for rows.Next() {
		v, err := s.scanVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("funcstore: version list scan: %w", err)
		}
		versions = append(versions, v)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("funcstore: version list rows: %w", err)
	}
	return versions, nil
}

func (s *storeSQL) FunctionVersionGet(ctx context.Context, versionID string) (FunctionVersionInterface, error) {
	if versionID == "" {
		return nil, ErrInvalidID
	}

	q := fmt.Sprintf("SELECT id, function_id, source, created_at FROM %s WHERE id = %s",
		s.versionsTableName, s.placeholder(1))

	row := s.db.QueryRowContext(ctx, q, versionID)
	v, err := s.scanVersion(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("funcstore: version get: %w", err)
	}
	return v, nil
}

func (s *storeSQL) FunctionVersionDelete(ctx context.Context, versionID string) error {
	if versionID == "" {
		return ErrInvalidID
	}

	q := fmt.Sprintf("DELETE FROM %s WHERE id = %s", s.versionsTableName, s.placeholder(1))
	res, err := s.db.ExecContext(ctx, q, versionID)
	if err != nil {
		return fmt.Errorf("funcstore: version delete: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *storeSQL) FunctionRestore(ctx context.Context, functionID string, versionID string) error {
	if functionID == "" || versionID == "" {
		return ErrInvalidID
	}

	// Load the target version.
	version, err := s.FunctionVersionGet(ctx, versionID)
	if err != nil {
		return err
	}
	if version.GetFunctionID() != functionID {
		return errors.New("funcstore: version does not belong to the specified function")
	}

	// Load the current function and snapshot its source so the current
	// state is also recoverable.
	fn, err := s.FunctionFindByID(ctx, functionID)
	if err != nil {
		return err
	}
	if err := s.snapshotVersion(ctx, functionID, fn.GetSource()); err != nil {
		return fmt.Errorf("funcstore: snapshot before restore: %w", err)
	}

	// Set the source to the version's source and update directly (bypassing
	// the auto-snapshot in FunctionUpdate, since we already snapshotted above).
	fn.SetSource(version.GetSource())
	fn.SetUpdatedAt(strconv.FormatInt(time.Now().Unix(), 10))

	q := fmt.Sprintf("UPDATE %s SET source = %s, updated_at = %s WHERE id = %s",
		s.tableName,
		s.placeholder(1), s.placeholder(2), s.placeholder(3),
	)

	res, err := s.db.ExecContext(ctx, q, fn.GetSource(), fn.GetUpdatedAt(), fn.GetID())
	if err != nil {
		return fmt.Errorf("funcstore: restore: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *storeSQL) DB() *sql.DB {
	return s.db
}

// === Internal ================================================================

func (s *storeSQL) resolve(ctx context.Context, key string) (FunctionInterface, error) {
	// Try ID first.
	fn, err := s.FunctionFindByID(ctx, key)
	if err == nil {
		return fn, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	// Fall back to alias (path or leaf).
	fn, err = s.FunctionFindByAlias(ctx, key)
	if err != nil {
		return nil, err
	}
	return fn, nil
}

func (s *storeSQL) getRuntime(name string) Runtime {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtimes[name]
}

func (s *storeSQL) findGroup(ctx context.Context, key string) (GroupInterface, error) {
	if key == "" {
		return nil, ErrGroupNotFound
	}
	g, err := s.GroupFindByPath(ctx, key)
	if err == nil {
		return g, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return s.GroupFindByID(ctx, key)
}

func (s *storeSQL) resolveGroupForFunction(ctx context.Context, fn FunctionInterface) (GroupInterface, error) {
	groupKey := strings.TrimSpace(fn.GetGroup())
	if groupKey == "" && strings.TrimSpace(fn.GetGroupID()) != "" {
		groupKey = fn.GetGroupID()
	}
	if groupKey == "" {
		return s.ensureDefaultGroup(ctx)
	}
	g, err := s.findGroup(ctx, groupKey)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	return g, nil
}

func (s *storeSQL) validateAndSetAliasPath(ctx context.Context, fn FunctionInterface, ownID string) error {
	if fn.GetAlias() == "" {
		fn.SetAliasPath("")
		return nil
	}
	if strings.Contains(fn.GetAlias(), "/") {
		return errors.New("funcstore: alias cannot contain '/'")
	}

	group, err := s.GroupFindByID(ctx, fn.GetGroupID())
	if err != nil {
		return err
	}

	aliasPath := group.GetPath() + "/" + fn.GetAlias()
	fn.SetAliasPath(aliasPath)

	q := fmt.Sprintf("SELECT id FROM %s WHERE group_id = %s AND alias = %s", s.tableName, s.placeholder(1), s.placeholder(2))
	var existingID string
	err = s.db.QueryRowContext(ctx, q, fn.GetGroupID(), fn.GetAlias()).Scan(&existingID)
	if err == nil && existingID != ownID {
		return ErrAliasTaken
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("funcstore: check alias: %w", err)
	}
	return nil
}

func (s *storeSQL) ensureDefaultGroup(ctx context.Context) (GroupInterface, error) {
	g, err := s.GroupFindByPath(ctx, "default")
	if err == nil {
		return g, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	g = NewGroup()
	g.SetSlug("default").SetPath("default")
	q := fmt.Sprintf(`INSERT INTO %s (id, slug, parent_id, path, description, metadata, created_at, updated_at) VALUES (%s)`,
		s.groupsTableName, s.placeholders(8))
	_, err = s.db.ExecContext(ctx, q,
		g.GetID(), g.GetSlug(), nil, g.GetPath(), nil, nil, g.GetCreatedAt(), g.GetUpdatedAt(),
	)
	if err != nil {
		return nil, fmt.Errorf("funcstore: create default group: %w", err)
	}
	return g, nil
}

func (s *storeSQL) runWithTimeout(ctx context.Context, prog Program, input any, timeout time.Duration) (any, error) {
	type result struct {
		out any
		err error
	}

	done := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{nil, fmt.Errorf("%w: %v", ErrPanic, r)}
			}
		}()
		out, err := prog.Run(input)
		done <- result{out, err}
	}()

	// If the program supports interruption (e.g. the goja runtime), stop the
	// abandoned goroutine instead of letting it run forever. Runtimes that
	// cannot be interrupted still leak their goroutine — a fundamental
	// limitation of in-process execution.
	interrupt := func() {
		if ip, ok := prog.(interface{ Interrupt() }); ok {
			ip.Interrupt()
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		interrupt()
		return nil, ErrTimeout
	case <-ctx.Done():
		interrupt()
		return nil, ctx.Err()
	case res := <-done:
		return res.out, res.err
	}
}

// === Helpers ===

func (s *storeSQL) placeholder(pos int) string {
	if isPostgres(s.driver) {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func (s *storeSQL) placeholders(n int) string {
	if isPostgres(s.driver) {
		parts := make([]string, n)
		for i := 0; i < n; i++ {
			parts[i] = fmt.Sprintf("$%d", i+1)
		}
		return strings.Join(parts, ", ")
	}
	return strings.Repeat("?, ", n-1) + "?"
}

func isPostgres(driver string) bool {
	switch driver {
	case "postgres", "pgx", "postgresql":
		return true
	default:
		return false
	}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// escapeLike escapes the LIKE wildcard characters in s so it can be embedded
// in a LIKE pattern with ESCAPE '\'. Without this, group slugs containing
// '_' or '%' would match unrelated paths.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *storeSQL) scanFunction(sc scanner) (FunctionInterface, error) {
	var id, groupID, runtime, source string
	var alias, aliasPath, name, metadata sql.NullString
	var timeoutMs, memLimit, createdAt, updatedAt int64

	err := sc.Scan(
		&id, &groupID, &alias, &aliasPath, &name, &runtime, &source,
		&timeoutMs, &memLimit, &metadata,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	data := map[string]string{
		COLUMN_ID:         id,
		COLUMN_GROUP_ID:   groupID,
		COLUMN_ALIAS:      alias.String,
		COLUMN_ALIAS_PATH: aliasPath.String,
		COLUMN_NAME:       name.String,
		COLUMN_RUNTIME:    runtime,
		COLUMN_SOURCE:     source,
		COLUMN_TIMEOUT_MS: strconv.FormatInt(timeoutMs, 10),
		COLUMN_MEM_LIMIT:  strconv.FormatInt(memLimit, 10),
		COLUMN_METADATA:   metadata.String,
		COLUMN_CREATED_AT: strconv.FormatInt(createdAt, 10),
		COLUMN_UPDATED_AT: strconv.FormatInt(updatedAt, 10),
	}

	fn := NewFunction()
	fn.Hydrate(data)
	return fn, nil
}

func (s *storeSQL) scanGroup(sc scanner) (GroupInterface, error) {
	var id, slug, path string
	var parentID, description, metadata sql.NullString
	var createdAt, updatedAt int64

	err := sc.Scan(
		&id, &slug, &parentID, &path, &description, &metadata,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	data := map[string]string{
		COLUMN_GROUP_ID_G:        id,
		COLUMN_GROUP_SLUG:        slug,
		COLUMN_GROUP_PARENT_ID:   parentID.String,
		COLUMN_GROUP_PATH:        path,
		COLUMN_GROUP_DESCRIPTION: description.String,
		COLUMN_GROUP_METADATA:    metadata.String,
		COLUMN_GROUP_CREATED_AT:  strconv.FormatInt(createdAt, 10),
		COLUMN_GROUP_UPDATED_AT:  strconv.FormatInt(updatedAt, 10),
	}

	g := NewGroup()
	g.Hydrate(data)
	return g, nil
}

// snapshotVersion inserts a source snapshot into the versions table.
func (s *storeSQL) snapshotVersion(ctx context.Context, functionID string, source string) error {
	v := NewFunctionVersion()
	v.SetFunctionID(functionID).SetSource(source)

	q := fmt.Sprintf(`INSERT INTO %s (id, function_id, source, created_at) VALUES (%s)`,
		s.versionsTableName, s.placeholders(4))

	_, err := s.db.ExecContext(ctx, q,
		v.GetID(), v.GetFunctionID(), v.GetSource(), v.GetCreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("funcstore: insert version: %w", err)
	}
	return nil
}

func (s *storeSQL) scanVersion(sc scanner) (FunctionVersionInterface, error) {
	var id, functionID, source string
	var createdAt int64

	err := sc.Scan(&id, &functionID, &source, &createdAt)
	if err != nil {
		return nil, err
	}

	data := map[string]string{
		COLUMN_VERSION_ID:          id,
		COLUMN_VERSION_FUNCTION_ID: functionID,
		COLUMN_VERSION_SOURCE:      source,
		COLUMN_VERSION_CREATED_AT:  strconv.FormatInt(createdAt, 10),
	}

	v := NewFunctionVersion()
	v.Hydrate(data)
	return v, nil
}
