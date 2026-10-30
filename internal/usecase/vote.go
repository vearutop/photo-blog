package usecase

import (
	"context"
	"errors"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/storage"
)

type voteDeps interface {
	VoteRepository() *storage.VoteRepository
	PhotoAlbumFinder() uniq.Finder[photo.Album]
}

type voteInput struct {
	AlbumName string    `query:"album_name" required:"true"`
	ImageHash uniq.Hash `query:"image_hash"`
}

type voteState struct {
	Open     bool                `json:"open"`
	MaxPicks int                 `json:"max_picks"`
	Picks    []uniq.Hash         `json:"picks"`
	Results  []storage.VoteCount `json:"results,omitempty" description:"Most picked first, only when results are visible."`
	Voters   int                 `json:"voters,omitempty"`
	CanSee   bool                `json:"can_see_results"`
}

// voteAlbum finds the album and visitor of a vote request.
func voteAlbum(ctx context.Context, deps voteDeps, albumName string) (photo.Album, uniq.Hash, error) {
	visitorHash := auth.VisitorFromContext(ctx)
	if visitorHash == 0 {
		return photo.Album{}, 0, errors.New("missing visitor hash")
	}

	album, err := deps.PhotoAlbumFinder().FindByHash(ctx, uniq.StringHash(albumName))
	if err != nil {
		return album, 0, err
	}

	if !album.Settings.Vote.Enabled {
		return album, 0, status.Wrap(errors.New("voting is not enabled for this album"), status.FailedPrecondition)
	}

	return album, visitorHash, nil
}

// GetVote returns visitor picks and, if visible, results of album voting.
func GetVote(deps voteDeps) usecase.Interactor {
	return usecase.NewInteractor(func(ctx context.Context, in voteInput, out *voteState) error {
		album, visitorHash, err := voteAlbum(ctx, deps, in.AlbumName)
		if err != nil {
			return err
		}

		v := album.Settings.Vote
		out.Open = v.Open
		out.MaxPicks = v.Limit()
		out.CanSee = v.ResultsPublic || auth.IsAdmin(ctx)

		if out.Picks, err = deps.VoteRepository().Picks(ctx, album.Hash, visitorHash); err != nil {
			return err
		}

		if out.CanSee {
			out.Results, out.Voters, err = deps.VoteRepository().Results(ctx, album.Hash)
		}

		return err
	})
}

// AddVote picks an image in album voting.
func AddVote(deps voteDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in voteInput, out *struct{}) error {
		album, visitorHash, err := voteAlbum(ctx, deps, in.AlbumName)
		if err != nil {
			return err
		}

		if !album.Settings.Vote.Open {
			return status.Wrap(errors.New("voting is closed"), status.FailedPrecondition)
		}

		if ok, err := deps.VoteRepository().InAlbum(ctx, album.Hash, in.ImageHash); err != nil {
			return err
		} else if !ok {
			return status.Wrap(errors.New("image is not in album"), status.InvalidArgument)
		}

		// ponytail: count-then-insert races, a double click may exceed the limit by one, fraud protection is out of scope.
		picks, err := deps.VoteRepository().Picks(ctx, album.Hash, visitorHash)
		if err != nil {
			return err
		}

		for _, h := range picks {
			if h == in.ImageHash {
				return nil
			}
		}

		if len(picks) >= album.Settings.Vote.Limit() {
			return status.Wrap(errors.New("no picks left"), status.FailedPrecondition)
		}

		return deps.VoteRepository().Add(ctx, album.Hash, visitorHash, in.ImageHash)
	})
	u.SetExpectedErrors(status.FailedPrecondition, status.InvalidArgument)

	return u
}

// DeleteVote removes a pick in album voting.
func DeleteVote(deps voteDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, in voteInput, out *struct{}) error {
		album, visitorHash, err := voteAlbum(ctx, deps, in.AlbumName)
		if err != nil {
			return err
		}

		if !album.Settings.Vote.Open {
			return status.Wrap(errors.New("voting is closed"), status.FailedPrecondition)
		}

		return deps.VoteRepository().Delete(ctx, album.Hash, visitorHash, in.ImageHash)
	})
	u.SetExpectedErrors(status.FailedPrecondition)

	return u
}
