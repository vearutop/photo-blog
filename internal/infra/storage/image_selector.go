package storage

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/domain/photo"
)

type ImageSelector struct {
	st *sqluct.Storage

	ai sqluct.StorageOf[AlbumImage]
	a  sqluct.StorageOf[photo.Album]
	i  sqluct.StorageOf[photo.Image]
	m  sqluct.StorageOf[photo.Meta]
	e  sqluct.StorageOf[photo.Exif]

	ref *sqluct.Referencer
}

func NewImageSelector(storage *sqluct.Storage) *ImageSelector {
	f := &ImageSelector{
		st: storage,
		ai: sqluct.Table[AlbumImage](storage, AlbumImageTable),
		a:  sqluct.Table[photo.Album](storage, AlbumTable),
		i:  sqluct.Table[photo.Image](storage, ImageTable),
		m:  sqluct.Table[photo.Meta](storage, MetaTable),
		e:  sqluct.Table[photo.Exif](storage, ExifTable),
	}

	ref := storage.MakeReferencer()

	ref.AddTableAlias(f.ai.R, AlbumImageTable)
	ref.AddTableAlias(f.a.R, AlbumTable)
	ref.AddTableAlias(f.m.R, MetaTable)
	ref.AddTableAlias(f.i.R, ImageTable)
	ref.AddTableAlias(f.e.R, ExifTable)

	f.ref = ref

	return f
}

type ImageQuery struct {
	f *ImageSelector
	q squirrel.SelectBuilder

	albumJoined       bool
	albumImagesJoined bool
	metaJoined        bool
	exifJoined        bool
	withSingleAlbum   bool
}

func (r *ImageSelector) Select() *ImageQuery {
	q := r.i.SelectStmt()

	return &ImageQuery{f: r, q: q}
}

func (is *ImageQuery) joinAlbumImages() *ImageQuery {
	if is.albumImagesJoined {
		return is
	}

	is.albumImagesJoined = true
	ref := is.f.ref
	air := is.f.ai.R
	ir := is.f.i.R
	is.q = is.q.LeftJoin(ref.Fmt("%s ON %s = %s", air, &air.ImageHash, &ir.Hash)).GroupBy(ref.Ref(&ir.Hash))

	return is
}

func (is *ImageQuery) joinAlbums() *ImageQuery {
	if is.albumJoined {
		return is
	}

	is.joinAlbumImages()

	is.albumJoined = true
	ref := is.f.ref
	air := is.f.ai.R
	ar := is.f.a.R
	is.q = is.q.LeftJoin(ref.Fmt("%s ON %s = %s", ar, &ar.Hash, &air.AlbumHash))

	return is
}

func (is *ImageQuery) joinExif() *ImageQuery {
	if is.exifJoined {
		return is
	}

	is.exifJoined = true
	ref := is.f.ref
	ir := is.f.i.R
	er := is.f.e.R
	is.q = is.q.LeftJoin(ref.Fmt("%s ON %s = %s", er, &er.Hash, &ir.Hash))

	return is
}

func (is *ImageQuery) joinMeta() *ImageQuery {
	if is.metaJoined {
		return is
	}

	is.metaJoined = true
	ref := is.f.ref
	ir := is.f.i.R
	mr := is.f.m.R
	is.q = is.q.LeftJoin(ref.Fmt("%s ON %s = %s", mr, &mr.Hash, &ir.Hash))

	return is
}

func (is *ImageQuery) OnlyPublic() *ImageQuery {
	is.joinAlbums()

	ref := is.f.ref
	ar := is.f.a.R

	is.q = is.q.Where(squirrel.Eq{ref.Ref(&ar.Public): true})

	return is
}

func (is *ImageQuery) ByAlbumName(albumName string) *ImageQuery {
	if is.withSingleAlbum {
		panic("single album is already set")
	}

	is.withSingleAlbum = true
	h := photo.AlbumHash(albumName)

	ref := is.f.ref
	air := is.f.ai.R

	is.joinAlbumImages()
	is.q = is.q.Where(squirrel.Eq{ref.Ref(&air.AlbumHash): h})

	return is
}

func (is *ImageQuery) Search(query string) *ImageQuery {
	is.joinMeta()

	ref := is.f.ref
	mr := is.f.m.R

	is.q = is.q.Where(ref.Fmt("%s LIKE ?", &mr.Data), "%"+query+"%")

	return is
}

func (is *ImageQuery) ByLens(lens string) *ImageQuery {
	is.joinExif()

	ref := is.f.ref
	er := is.f.e.R

	is.q = is.q.Where(ref.Fmt("%s LIKE ?", &er.LensModel), "%"+lens+"%")

	return is
}

func (is *ImageQuery) ByCamera(camera string) *ImageQuery {
	is.joinExif()

	ref := is.f.ref
	er := is.f.e.R

	is.q = is.q.Where(ref.Fmt("%s LIKE ?", &er.CameraModel), "%"+camera+"%")

	return is
}

func (is *ImageQuery) order() *ImageQuery {
	ref := is.f.ref
	ir := is.f.i.R
	air := is.f.ai.R

	if is.withSingleAlbum {
		is.q = is.q.OrderByClause(ref.Fmt("COALESCE(%s, %s, %s), %s", &air.Timestamp, &ir.TakenAt, &ir.CreatedAt, &ir.Path))
	} else {
		is.q = is.q.OrderByClause(ref.Fmt("COALESCE(%s, %s), %s", &ir.TakenAt, &ir.CreatedAt, &ir.Path))
	}

	return is
}

func (is *ImageQuery) Limit(lim uint64) *ImageQuery {
	is.q = is.q.Limit(lim)

	return is
}

func (is *ImageQuery) Offset(off uint64) *ImageQuery {
	is.q = is.q.Offset(off)

	return is
}

func (is *ImageQuery) Find(ctx context.Context) ([]photo.Image, error) {
	is.order()

	return is.f.i.List(ctx, is.q)
}
