# Sprite Management

This package builds and serves reusable thumbnail sprites for:

- album gallery thumbs
- sub-album preview thumbs
- map marker thumbs

The current implementation separates two concerns:

1. sprite chunk blobs (JPEGs in blob storage)
2. sprite manifests (JSON, mapping image hashes to chunk placement)

## Goals

- keep page-load thumbnail requests low
- reuse identical sprite chunks across different manifests
- retire unused manifests and chunks without an unbounded storage leak
- self-heal a chunk that goes missing instead of 404ing forever

## Terms

`chunk`
: A JPEG sprite image stored in blob storage, e.g. `/thumb-sprite/album-sprite:....jpg`.

`manifest`
: A JSON structure that maps image hashes to chunk ids and placement metadata, plus the
owning albums and the build plan behind each chunk (see [Chunk Plans](#chunk-plans-and-self-heal)).

## Identity Model

### Chunk key

Chunk keys are content-derived. They include:

- sprite service version
- scale
- layout bucket width
- thumbnail source size
- compose mode
- exact ordered image list in that chunk
- source image dimensions
- computed rendered heights

This means:

- identical chunk inputs reuse the same blob
- different manifests can still share the same chunk blobs

### Manifest key

Manifest keys are also content-derived, built from the exact ordered sprite input list:

- visible main album images
- visible sub-album preview images merged into the page
- image hash, width, height
- whether a GPS marker sprite is needed for that image

Album identity is not part of the manifest key. If two albums produce the same sprite
input plan (e.g. identical image sets), they reuse the same manifest and its chunks.

## Ownership And Retirement

Each `Manifest` carries an `Albums []uniq.Hash` list of the albums currently relying on
it. This is the actual reference count, kept as a field on the manifest itself rather
than in a separate table.

- `TrackAlbum(ctx, images, albumHash)` adds an owner if not already present, and returns
  a stable `retirementKey` (`manifestKey|albumHash`) for that ownership claim.
- `Delete(ctx, retirementKey)` removes that album from `manifest.Albums`. If other owners
  remain, the manifest is written back as-is. If none remain, the manifest is written
  back ownerless and `scheduleRetirement` is armed.
- `scheduleRetirement` waits `retirementDelay` (default 5 minutes) and then re-checks: if
  the manifest is *still* ownerless, `deleteManifest` removes the manifest record and any
  of its chunks that aren't referenced by some other still-stored manifest
  (`manifestChunkRefsExcluding`). The delay exists to avoid thrashing on rapid edits and
  metadata-only changes that don't actually need a rebuild.

`Delete` is wired as the eviction hook for the synthetic `sprite-retire` cache name in
the app's invalidation index (see `internal/infra/dep/README.md`, section "Album Sprite
Retirement") — an album/sub-album content change is what triggers ownership loss, not a
TTL on the manifest itself.

### Chunk pinning (concurrency guard)

`ensureChunk` does a read-check-then-write: if a chunk blob already exists, it's reused
as-is (that's the whole point of content-derived keys). This creates a narrow race
against `deleteManifest`, which decides what's safe to delete by walking *persisted*
manifests: a manifest that's mid-build (chunk reused, but not yet written back) is
invisible to that walk, so a concurrent retirement could delete a chunk a moment before
the new manifest referencing it gets persisted.

`build()`/`buildMarkerSprites()` close this by pinning every chunk key the instant they
decide to reuse or create it (`pinChunk`), and `EnsureBuild` only releases the pins after
the manifest is actually persisted. `deleteManifest` skips deleting any currently pinned
chunk, deferring it to the next retirement pass instead of racing it.

## Chunk Plans And Self-Heal

Each `Manifest.ChunkPlans` entry records the exact bucket/scale/mode/image-list that
produced a given chunk key. Because chunk keys are content-derived, replaying that plan
reproduces the identical blob.

If a chunk blob is missing when requested (lost to a race that slipped past the guard
above, an operator mistake, a storage hiccup, or anything else), `ShowAlbumSprite` calls
`Service.RegenerateChunk(ctx, key)` as a fallback before returning 404:

1. walk stored manifests for one with a `ChunkPlans` entry for that key
2. pin the key and rebuild it via the same `ensureChunk` used during normal builds
3. return the freshly written blob

Concurrent `RegenerateChunk` calls for the *same* missing key (the common case — many
page loads referencing the same chunk) are deduplicated with a `singleflight.Group`, so
one missing chunk triggers one rebuild, not one per concurrent request.

Manifests persisted before `ChunkPlans` existed simply have no plan and fall through to
a real 404, same as before this mechanism was added — they get replaced by a fresh
manifest on the next content change regardless.

## Manual Cleanup

`internal/usecase/control/integrity/cleanup_album_sprites.go` (`POST
/cleanup-album-sprites`) is an admin tool, not part of automatic retirement. It
cross-references actually-cached album pages against stored manifests and chunks to find
and optionally delete things automatic retirement missed — e.g. leftover manifests from
before some other bug, or blobs orphaned by a failed partial build. Run it with
`dry_run=true` first to see what it would touch.

## Why This Works

- content-derived manifest keys let identical outputs reuse manifests
- content-derived chunk keys let related manifests share actual sprite JPEGs
- ownership tracked directly on the manifest keeps retirement simple: no owners, no need
- the delay absorbs rapid, low-value churn without leaving chunks live forever
- chunk pinning and self-heal-on-404 mean a lost or racily-deleted chunk is a rare,
  self-correcting event rather than a permanent broken image

## Current Constraints

- retirement depends on album/sub-album invalidation events reaching the right manifest;
  see `internal/infra/dep/README.md` for how those labels are attached
- the 5-minute retirement delay is shorter than the 30-hour album-page cache TTL and the
  1-year `Cache-Control` sprites are served with — a page or CDN edge holding a
  superseded manifest's chunk URL for longer than that window will 404 on first request,
  then self-heal via `RegenerateChunk` on the retry
- self-heal only works for manifests built after `ChunkPlans` was introduced

## Related Code

- [service.go](service.go)
- [../../../usecase/show_album_sprite.go](../../../usecase/show_album_sprite.go)
- [../../../usecase/control/integrity/cleanup_album_sprites.go](../../../usecase/control/integrity/cleanup_album_sprites.go)
- [../../dep/cache.go](../../dep/cache.go)
- [../../dep/README.md](../../dep/README.md)
