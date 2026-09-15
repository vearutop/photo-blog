package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bool64/brick/database"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/bool64/stats"
	"github.com/stretchr/testify/require"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	sqlitemigrations "github.com/vearutop/photo-blog/internal/infra/storage/sqlite"
)

func TestImageRepositoryEnsure_PopulatesUTimeFromTakenAt(t *testing.T) {
	t.Parallel()

	repo := NewImageRepository(testImageStorage(t))
	takenAt := time.Date(2024, time.February, 3, 4, 5, 6, 0, time.UTC)

	img, err := repo.Ensure(context.Background(), photo.Image{
		File: uniq.File{
			Head: uniq.Head{
				Time: uniq.Time{
					CreatedAt: time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC),
				},
				Hash: 101,
			},
			Path: "taken-at.jpg",
		},
		TakenAt: &takenAt,
	})
	require.NoError(t, err)
	require.Equal(t, takenAt.Unix(), img.UTime)

	stored, err := repo.FindByHash(context.Background(), img.Hash)
	require.NoError(t, err)
	require.Equal(t, takenAt.Unix(), stored.UTime)
}

func TestImageRepositoryEnsure_PopulatesUTimeFromCreatedAt(t *testing.T) {
	t.Parallel()

	repo := NewImageRepository(testImageStorage(t))
	createdAt := time.Date(2024, time.March, 4, 5, 6, 7, 0, time.UTC)

	img, err := repo.Ensure(context.Background(), photo.Image{
		File: uniq.File{
			Head: uniq.Head{
				Time: uniq.Time{
					CreatedAt: createdAt,
				},
				Hash: 202,
			},
			Path: "created-at.jpg",
		},
	})
	require.NoError(t, err)
	require.NotZero(t, img.CreatedAt)
	require.Equal(t, img.CreatedAt.Unix(), img.UTime)

	stored, err := repo.FindByHash(context.Background(), img.Hash)
	require.NoError(t, err)
	require.Equal(t, stored.CreatedAt.Unix(), stored.UTime)
}

func TestImageRepositoryEnsure_IgnoresCustomUTime(t *testing.T) {
	t.Parallel()

	repo := NewImageRepository(testImageStorage(t))
	takenAt := time.Date(2024, time.April, 5, 6, 7, 8, 0, time.UTC)

	img, err := repo.Ensure(context.Background(), photo.Image{
		File: uniq.File{
			Head: uniq.Head{
				Hash: 303,
			},
			Path: "custom-utime.jpg",
		},
		TakenAt: &takenAt,
		UTime:   1234567890,
	})
	require.NoError(t, err)
	require.Equal(t, takenAt.Unix(), img.UTime)

	stored, err := repo.FindByHash(context.Background(), img.Hash)
	require.NoError(t, err)
	require.Equal(t, takenAt.Unix(), stored.UTime)
}

func TestImageRepositoryEnsure_UpdateHonorsTakenAtForUTime(t *testing.T) {
	t.Parallel()

	repo := NewImageRepository(testImageStorage(t))
	createdAt := time.Date(2024, time.May, 6, 7, 8, 9, 0, time.UTC)
	takenAt := time.Date(2024, time.June, 7, 8, 9, 10, 0, time.UTC)

	original, err := repo.Ensure(context.Background(), photo.Image{
		File: uniq.File{
			Head: uniq.Head{
				Time: uniq.Time{
					CreatedAt: createdAt,
				},
				Hash: 404,
			},
			Path: "update-taken-at.jpg",
		},
	})
	require.NoError(t, err)
	require.NotZero(t, original.CreatedAt)
	require.Equal(t, original.CreatedAt.Unix(), original.UTime)

	updated, err := repo.Ensure(context.Background(), photo.Image{
		File: uniq.File{
			Head: uniq.Head{
				Hash: 404,
			},
			Path: "update-taken-at.jpg",
		},
		TakenAt: &takenAt,
		UTime:   1,
	})
	require.NoError(t, err)
	require.Equal(t, takenAt.Unix(), updated.UTime)

	stored, err := repo.FindByHash(context.Background(), updated.Hash)
	require.NoError(t, err)
	require.Equal(t, takenAt.Unix(), stored.UTime)
	require.NotNil(t, stored.TakenAt)
	require.True(t, stored.TakenAt.Equal(takenAt))
}

func TestImageRepositoryUpdate_IgnoresCustomUTime(t *testing.T) {
	t.Parallel()

	repo := NewImageRepository(testImageStorage(t))
	takenAt := time.Date(2024, time.July, 8, 9, 10, 11, 0, time.UTC)

	original, err := repo.Ensure(context.Background(), photo.Image{
		File: uniq.File{
			Head: uniq.Head{
				Hash: 505,
			},
			Path: "update-custom-utime.jpg",
		},
	})
	require.NoError(t, err)

	original.TakenAt = &takenAt
	original.UTime = 42

	err = repo.Update(context.Background(), original)
	require.NoError(t, err)

	stored, err := repo.FindByHash(context.Background(), original.Hash)
	require.NoError(t, err)
	require.Equal(t, takenAt.Unix(), stored.UTime)
	require.NotNil(t, stored.TakenAt)
	require.True(t, stored.TakenAt.Equal(takenAt))
}

func testImageStorage(t *testing.T) *sqluct.Storage {
	t.Helper()

	cfg := database.Config{
		DriverName:      "sqlite",
		DSN:             filepath.Join(t.TempDir(), "image.sqlite") + "?_time_format=sqlite",
		ApplyMigrations: true,
		MaxOpen:         1,
		MaxIdle:         1,
	}

	st, err := database.SetupStorageDSN(cfg, ctxd.NoOpLogger{}, stats.NoOp{}, sqlitemigrations.Migrations)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, st.DB().DB.Close())
	})

	return st
}
