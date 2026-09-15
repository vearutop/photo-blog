package integrity

import (
	"context"
	"fmt"
	"slices"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	rootusecase "github.com/vearutop/photo-blog/internal/usecase"
)

// persistentCacheNames references rootusecase's exported constants (rather than duplicating the
// literal names) so this list can't drift from what NewAlbumPageBuilder actually writes to. The
// enum tag on invalidatePersistentCacheInput.CacheName below can't do the same (Go struct tags
// must be literal strings) — keep it in sync by hand when albumCacheVersion changes.
var persistentCacheNames = []string{
	rootusecase.AlbumDataCacheName,
	rootusecase.AlbumPageCacheName,
	"main-page",
	"thumb-grid",
}

type invalidatePersistentCacheDeps interface {
	CtxdLogger() ctxd.Logger
	PersistentCacheStorage() *sqluct.Storage
}

type invalidatePersistentCacheInput struct {
	CacheName string `path:"cache_name" required:"true" enum:"album-data-2,album-page-2,main-page,thumb-grid" description:"Persistent cache to invalidate."`
}

type InvalidatePersistentCacheReport struct {
	CacheName          string `json:"cache_name"`
	DeletedRecordCount int    `json:"deleted_record_count"`
	DeletedLabelCount  int    `json:"deleted_label_count"`
}

func InvalidatePersistentCache(deps invalidatePersistentCacheDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in invalidatePersistentCacheInput, out *InvalidatePersistentCacheReport) error {
		if !slices.Contains(persistentCacheNames, in.CacheName) {
			return status.Wrap(fmt.Errorf("unsupported cache_name %q, expected one of %v", in.CacheName, persistentCacheNames), status.InvalidArgument)
		}

		report, err := invalidatePersistentCache(ctx, deps.PersistentCacheStorage(), in.CacheName)
		if err != nil {
			return err
		}

		deps.CtxdLogger().Info(ctx, "invalidate persistent cache",
			"cache_name", report.CacheName,
			"deleted_record_count", report.DeletedRecordCount,
			"deleted_label_count", report.DeletedLabelCount)

		*out = report

		return nil
	})

	u.SetTags("Control Panel", "Integrity")
	u.SetExpectedErrors(status.Unknown, status.InvalidArgument)
	u.SetDescription("Invalidates one persistent cache and its invalidation labels.")

	return u
}

func invalidatePersistentCache(ctx context.Context, st *sqluct.Storage, cacheName string) (InvalidatePersistentCacheReport, error) {
	tx, err := st.DB().DB.BeginTx(ctx, nil)
	if err != nil {
		return InvalidatePersistentCacheReport{}, fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	recordRes, err := tx.ExecContext(ctx, `DELETE FROM record WHERE cache_name = ?`, cacheName)
	if err != nil {
		return InvalidatePersistentCacheReport{}, fmt.Errorf("delete records for %s: %w", cacheName, err)
	}

	recordDeleted, err := recordRes.RowsAffected()
	if err != nil {
		return InvalidatePersistentCacheReport{}, fmt.Errorf("records affected for %s: %w", cacheName, err)
	}

	labelRes, err := tx.ExecContext(ctx, `DELETE FROM cache_label WHERE cache_name = ?`, cacheName)
	if err != nil {
		return InvalidatePersistentCacheReport{}, fmt.Errorf("delete labels for %s: %w", cacheName, err)
	}

	labelDeleted, err := labelRes.RowsAffected()
	if err != nil {
		return InvalidatePersistentCacheReport{}, fmt.Errorf("labels affected for %s: %w", cacheName, err)
	}

	if err := tx.Commit(); err != nil {
		return InvalidatePersistentCacheReport{}, fmt.Errorf("commit tx: %w", err)
	}
	committed = true

	return InvalidatePersistentCacheReport{
		CacheName:          cacheName,
		DeletedRecordCount: int(recordDeleted),
		DeletedLabelCount:  int(labelDeleted),
	}, nil
}
