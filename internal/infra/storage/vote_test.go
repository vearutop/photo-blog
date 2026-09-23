package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
)

func TestVoteRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := testImageStorage(t)
	repo := NewVoteRepository(st)

	const album, v1, v2 = uniq.Hash(1), uniq.Hash(10), uniq.Hash(20)

	require.NoError(t, NewAlbumRepository(st, NewImageRepository(st), NewMetaRepository(st)).AddImages(ctx, album, 100, 200))

	in, err := repo.InAlbum(ctx, album, 100)
	require.NoError(t, err)
	require.True(t, in)

	in, err = repo.InAlbum(ctx, album, 300)
	require.NoError(t, err)
	require.False(t, in)

	require.NoError(t, repo.Add(ctx, album, v1, 100))
	require.NoError(t, repo.Add(ctx, album, v1, 100)) // Repeated pick is a no-op.
	require.NoError(t, repo.Add(ctx, album, v1, 200))
	require.NoError(t, repo.Add(ctx, album, v2, 200))
	require.NoError(t, repo.Add(ctx, 2, v2, 100)) // Other album is not counted.

	picks, err := repo.Picks(ctx, album, v1)
	require.NoError(t, err)
	require.Equal(t, []uniq.Hash{100, 200}, picks)

	res, voters, err := repo.Results(ctx, album)
	require.NoError(t, err)
	require.Equal(t, 2, voters)
	require.Equal(t, []VoteCount{{ImageHash: 200, Count: 2}, {ImageHash: 100, Count: 1}}, res)

	require.NoError(t, repo.Delete(ctx, album, v1, 200))

	res, voters, err = repo.Results(ctx, album)
	require.NoError(t, err)
	require.Equal(t, 2, voters)
	require.Equal(t, []VoteCount{{ImageHash: 100, Count: 1}, {ImageHash: 200, Count: 1}}, res)
}
