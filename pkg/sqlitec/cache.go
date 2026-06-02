package sqlitec

import (
	"context"
	"database/sql"
	"errors"
	"runtime"
	"time"

	"github.com/bool64/cache"
	"github.com/bool64/sqluct"
)

var (
	_ cache.ReadWriterOf[any] = &dbMapOf[any]{}
	_ cache.Deleter           = &dbMapOf[any]{}
	_ cache.WalkerOf[any]     = &DBMapOf[any]{}
)

const (
	recordTable = "record"
)

type UnixTime int64

type record struct {
	CacheName string   `db:"cache_name" json:"cache_name,omitempty"`
	Key       string   `db:"key" json:"key,omitempty"`
	CreatedAt UnixTime `db:"created_at" json:"created_at,omitzero" title:"Created at"`
	UpdatedAt UnixTime `db:"updated_at" json:"updated_at,omitzero" title:"Updated at"`
	ExpireAt  UnixTime `db:"expire_at" json:"expire_at,omitzero" title:"Expire at"`
}

type RecordOf[V any] struct {
	record
	Val sqluct.JSON[V] `db:"val" json:"val,omizero" title:"Value"`
}

type RecordOfByte struct {
	record
	Val []byte `db:"val" json:"val,omizero" title:"Value"`
}

// DBMapOf is an in-memory cache backend. Please use NewDBMapOf to create it.
type DBMapOf[V any] struct {
	*dbMapOf[V]
}

type dbMapOf[V any] struct {
	*cache.InvalidationIndex

	cacheName string
	st        *sqluct.Storage

	t *cache.TraitOf[V]
}

// NewDBMapOf creates an instance of in-memory cache with optional configuration.
func NewDBMapOf[V any](st *sqluct.Storage, cacheName string, options ...func(cfg *cache.ConfigOf[V])) *DBMapOf[V] {
	c := &dbMapOf[V]{
		cacheName: cacheName,
		st:        st,
	}
	C := &DBMapOf[V]{
		dbMapOf: c,
	}

	cfg := cache.ConfigOf[V]{}
	for _, option := range options {
		option(&cfg)
	}

	evict := c.evictMostExpired

	c.t = cache.NewTraitOf[V](cfg.Policy, func(t *cache.Trait) {
		t.DeleteExpired = c.deleteExpired
		t.Len = c.Len
		t.Evict = evict
	})

	c.InvalidationIndex = cache.NewInvalidationIndex(c)

	runtime.SetFinalizer(C, func(m *DBMapOf[V]) {
		close(m.t.Closed)
	})

	return C
}

// Read gets value.
func (c *dbMapOf[V]) Read(ctx context.Context, key []byte) (val V, _ error) {
	if cache.SkipRead(ctx) {
		return val, cache.ErrNotFound
	}

	var err error

	cacheEntry := &cache.TraitEntryOf[V]{}
	found := true

	if _, ok := any(val).([]byte); ok {
		found, err = c.readBytes(ctx, string(key), any(cacheEntry).(*cache.TraitEntryOf[[]byte]))
	} else {
		found, err = c.readJSON(ctx, string(key), cacheEntry)
	}

	if err != nil {
		return val, err
	}

	if !found {
		cacheEntry = nil
		found = false
	}

	v, err := c.t.PrepareRead(ctx, cacheEntry, found)
	if err != nil {
		return val, err
	}

	return v, nil
}

func (c *dbMapOf[V]) readBytes(ctx context.Context, key string, cacheEntry *cache.TraitEntryOf[[]byte]) (found bool, err error) {
	r := RecordOfByte{}

	found = true

	qb := c.st.SelectStmt(recordTable, r).Where("cache_name = ?", c.cacheName).Where("key = ?", key)
	err = c.st.Select(ctx, qb, &r)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return found, err
		}

		found = false
	} else {
		cacheEntry.E = int64(time.Duration(r.ExpireAt) * time.Second)
		cacheEntry.K = []byte(r.Key)
		cacheEntry.V = r.Val
	}

	return found, nil
}

func (c *dbMapOf[V]) readJSON(ctx context.Context, key string, cacheEntry *cache.TraitEntryOf[V]) (found bool, err error) {
	r := RecordOf[V]{}

	found = true

	qb := c.st.SelectStmt(recordTable, r).Where("cache_name = ?", c.cacheName).Where("key = ?", key)
	err = c.st.Select(ctx, qb, &r)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return found, err
		}

		found = false
	} else {
		cacheEntry.E = int64(time.Duration(r.ExpireAt) * time.Second)
		cacheEntry.K = []byte(r.Key)
		cacheEntry.V = r.Val.Val
	}

	return found, nil
}

// Write sets value by the key.
func (c *dbMapOf[V]) Write(ctx context.Context, k []byte, v V) error {
	// Copy key to allow mutations of original argument.
	key := make([]byte, len(k))
	copy(key, k)

	ttl, expireAt := c.expireAt(ctx)

	var r any

	rec := record{}
	rec.CacheName = c.cacheName
	rec.Key = string(key)
	rec.CreatedAt = UnixTime(time.Now().Unix())
	rec.ExpireAt = UnixTime(time.Duration(expireAt) / time.Second)

	if b, ok := any(v).([]byte); ok {
		var rr RecordOfByte
		rr.record = rec
		rr.Val = b

		r = rr
	} else {
		var rr RecordOf[V]
		rr.record = rec
		rr.Val.Val = v

		r = rr
	}

	_, err := c.st.InsertStmt(recordTable, r).
		Suffix("ON CONFLICT(cache_name, key) DO UPDATE SET expire_at = EXCLUDED.expire_at, val = EXCLUDED.val, updated_at = unixepoch()").
		ExecContext(ctx)
	if err != nil {
		return err
	}

	c.t.NotifyWritten(ctx, key, v, ttl)

	return nil
}

func (c *dbMapOf[V]) expireAt(ctx context.Context) (time.Duration, int64) {
	if ttl := c.t.TTL(ctx); ttl != 0 {
		return ttl, time.Now().Add(ttl).UnixNano()
	}

	return 0, 0
}

// Delete removes value by the key.
//
// It fails with ErrNotFound if key does not exist.
func (c *dbMapOf[V]) Delete(ctx context.Context, key []byte) error {
	res, err := c.st.DeleteStmt(recordTable).Where("cache_name = ?", c.cacheName).Where("key = ?", string(key)).ExecContext(ctx)
	if err != nil {
		return err
	}

	aff, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if aff == 0 {
		return cache.ErrNotFound
	}

	c.t.NotifyDeleted(ctx, key)

	return nil
}

func (c *DBMapOf[V]) Walk(cb func(entry cache.EntryOf[V]) error) (int, error) {
	rows, err := c.st.DB().DB.QueryContext(context.Background(), `SELECT key, val, expire_at FROM record WHERE cache_name = ?`, c.cacheName)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var n int

	for rows.Next() {
		var rec RecordOf[V]
		if err := rows.Scan(&rec.Key, &rec.Val, &rec.ExpireAt); err != nil {
			return n, err
		}

		if err := cb(cache.TraitEntryOf[V]{
			K: []byte(rec.Key),
			V: rec.Val.Val,
			E: int64(time.Duration(rec.ExpireAt) * time.Second),
		}); err != nil {
			return n, err
		}

		n++
	}

	if err := rows.Err(); err != nil {
		return n, err
	}

	return n, nil
}

// ExpireAll marks all entries as expired, they can still serve stale cache.
func (c *dbMapOf[V]) ExpireAll(ctx context.Context) {
	start := time.Now()

	res, err := c.st.UpdateStmt(recordTable, nil).Set("expire_at", start.Unix()).Where("cache_name = ?", c.cacheName).ExecContext(ctx)
	if err != nil {
		println(err.Error()) // TODO: proper logging.
		return
	}

	aff, err := res.RowsAffected()
	if err != nil {
		println(err.Error())
		return
	}

	c.t.NotifyExpiredAll(ctx, start, int(aff))
}

// DeleteAll erases all entries.
func (c *dbMapOf[V]) DeleteAll(ctx context.Context) {
	start := time.Now()

	res, err := c.st.DeleteStmt(recordTable).Where("cache_name = ?", c.cacheName).ExecContext(ctx)
	if err != nil {
		println(err.Error())
		return
	}

	aff, err := res.RowsAffected()
	if err != nil {
		println(err.Error())
		return
	}

	c.t.NotifyDeletedAll(ctx, start, int(aff))
}

func (c *dbMapOf[V]) deleteExpired(before time.Time) {
	_, err := c.st.DeleteStmt(recordTable).Where("cache_name = ?", c.cacheName).Where("expire_at < ?", before.Unix()).Exec()
	if err != nil {
		println(err.Error())
		return
	}
}

// Len returns number of elements in cache.
func (c *dbMapOf[V]) Len() int {
	type Cnt struct {
		C int `db:"cnt"`
	}
	var cnt Cnt

	q := c.st.SelectStmt(recordTable, nil).Columns("COUNT(1) AS cnt").Where("cache_name = ?", c.cacheName)

	err := c.st.Select(context.Background(), q, &cnt)
	if err != nil {
		println(err.Error())
		return 0
	}

	return cnt.C
}

func (c *dbMapOf[V]) evictMostExpired(evictFraction float64) int {
	l := c.Len()

	if l == 0 {
		return 0
	}

	q := c.st.SelectStmt(recordTable, nil).
		Columns("key").
		Where("cache_name = ?", c.cacheName).
		OrderByClause("expire_at ASC").Limit(uint64(evictFraction * float64(l)))

	res, err := c.st.DeleteStmt(recordTable).Where("cache_name = ?", c.cacheName).Where("key IN (?)", q).Exec()
	if err != nil {
		println(err.Error())
		return 0
	}
	aff, err := res.RowsAffected()
	if err != nil {
		println(err.Error())
		return 0
	}
	return int(aff)
}
