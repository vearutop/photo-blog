package integrity

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/bool64/cache/filecache"
	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/image/sprite"
)

type cleanupAlbumSpritesDeps interface {
	CtxdLogger() ctxd.Logger
	PersistentCacheStorage() *sqluct.Storage
	AlbumSprites() *sprite.Service
}

type cleanupAlbumSpritesInput struct {
	DryRun bool `query:"dry_run" description:"Report cleanup candidates without deleting them."`
}

type AlbumSpriteCleanupReport struct {
	DryRun                    bool                   `json:"dry_run,omitempty"`
	BlobStoreRepair           filecache.RepairResult `json:"blob_store_repair"`
	AlbumPageCaches           int                    `json:"album_page_caches"`
	PageCachesWithSprites     int                    `json:"page_caches_with_sprites"`
	PageCachesWithoutSprites  int                    `json:"page_caches_without_sprites"`
	BrokenPageCacheKeys       []string               `json:"broken_page_cache_keys,omitempty"`
	BrokenPageCacheCount      int                    `json:"broken_page_cache_count"`
	WouldDeletePageCacheKeys  []string               `json:"would_delete_page_cache_keys,omitempty"`
	WouldDeletePageCacheCount int                    `json:"would_delete_page_cache_count"`
	DeletedPageCacheKeys      []string               `json:"deleted_page_cache_keys,omitempty"`
	DeletedPageCacheCount     int                    `json:"deleted_page_cache_count"`
	ReferencedManifestKeys    []string               `json:"referenced_manifest_keys,omitempty"`
	ReferencedManifestCount   int                    `json:"referenced_manifest_count"`
	ReferencedManifestSize    int64                  `json:"referenced_manifest_size"`
	MissingManifestReferences []string               `json:"missing_manifest_references,omitempty"`
	MissingManifestCount      int                    `json:"missing_manifest_count"`
	WouldDeleteManifestKeys   []string               `json:"would_delete_manifest_keys,omitempty"`
	WouldDeleteManifestCount  int                    `json:"would_delete_manifest_count"`
	DeletedManifestKeys       []string               `json:"deleted_manifest_keys,omitempty"`
	DeletedManifestCount      int                    `json:"deleted_manifest_count"`
	TotalBlobCount            int                    `json:"total_blob_count"`
	TotalBlobSize             int64                  `json:"total_blob_size"`
	ReferencedBlobCount       int                    `json:"referenced_blob_count"`
	ReferencedBlobSize        int64                  `json:"referenced_blob_size"`
	WouldDeleteBlobKeys       []string               `json:"would_delete_blob_keys,omitempty"`
	WouldDeleteBlobCount      int                    `json:"would_delete_blob_count"`
	WouldDeleteBlobSize       int64                  `json:"would_delete_blob_size"`
	DeletedBlobKeys           []string               `json:"deleted_blob_keys,omitempty"`
	DeletedBlobCount          int                    `json:"deleted_blob_count"`
	DeletedBlobSize           int64                  `json:"deleted_blob_size"`
}

type cachedJSONRecord struct {
	Key string `db:"key"`
	Val string `db:"val"`
}

type storedManifestRecord struct {
	Manifest sprite.Manifest
	Size     int64
}

type cachedAlbumPageRef struct {
	CacheKey    string
	ManifestKey string
	HasSprites  bool
}

type cachedAlbumPageData struct {
	SubAlbums         []cachedAlbumOutput     `json:"SubAlbums"`
	AlbumData         cachedAlbumOutput       `json:"AlbumData"`
	SpriteSheets      map[string]sprite.Sheet `json:"SpriteSheets"`
	SpriteManifestKey string                  `json:"SpriteManifestKey"`
}

type cachedAlbumOutput struct {
	Images []cachedImage `json:"Images"`
}

type cachedImage struct {
	Hash      string          `json:"Hash"`
	Width     int64           `json:"Width"`
	Height    int64           `json:"Height"`
	Gps       json.RawMessage `json:"Gps"`
	Is360Pano bool            `json:"Is360Pano"`
}

// CleanupAlbumSprites removes sprite manifests and chunk blobs that are no longer referenced by cached album pages.
func CleanupAlbumSprites(deps cleanupAlbumSpritesDeps) usecase.IOInteractorOf[cleanupAlbumSpritesInput, AlbumSpriteCleanupReport] {
	u := usecase.NewInteractor(func(ctx context.Context, in cleanupAlbumSpritesInput, out *AlbumSpriteCleanupReport) error {
		report, err := cleanupAlbumSprites(ctx, deps, in.DryRun)
		if err != nil {
			return err
		}

		*out = report

		return nil
	})

	u.SetTags("Control Panel", "Integrity")
	u.SetExpectedErrors(status.Unknown)

	return u
}

func cleanupAlbumSprites(ctx context.Context, deps cleanupAlbumSpritesDeps, dryRun bool) (AlbumSpriteCleanupReport, error) {
	blobStoreRepair := deps.AlbumSprites().RepairBlobStore(dryRun)

	pageRefs, pageKeys, pageCount, pageCachesWithSprites, err := cachedAlbumPageManifestKeys(ctx, deps.PersistentCacheStorage(), deps.AlbumSprites())
	if err != nil {
		return AlbumSpriteCleanupReport{}, err
	}

	manifests, err := storedSpriteManifests(ctx, deps.PersistentCacheStorage())
	if err != nil {
		return AlbumSpriteCleanupReport{}, err
	}
	manifestChunkRefs := allManifestChunkRefs(manifests)

	referencedChunks := make(map[string]struct{}, len(manifests))
	allManifestChunks := make(map[string]struct{}, len(manifests))
	missingRefs := make([]string, 0)

	for _, manifest := range manifests {
		for chunk := range manifestChunks(manifest.Manifest) {
			allManifestChunks[chunk] = struct{}{}
		}
	}

	for key := range pageKeys {
		manifest, ok := manifests[key]
		if !ok {
			missingRefs = append(missingRefs, key)
			continue
		}

		for chunk := range manifestChunks(manifest.Manifest) {
			referencedChunks[chunk] = struct{}{}
		}
	}

	report := AlbumSpriteCleanupReport{
		DryRun:                    dryRun,
		BlobStoreRepair:           blobStoreRepair,
		AlbumPageCaches:           pageCount,
		PageCachesWithSprites:     pageCachesWithSprites,
		PageCachesWithoutSprites:  pageCount - pageCachesWithSprites,
		ReferencedManifestKeys:    sortedKeys(pageKeys),
		ReferencedManifestCount:   len(pageKeys),
		MissingManifestReferences: missingRefs,
		MissingManifestCount:      len(missingRefs),
	}

	for _, pageRef := range pageRefs {
		if pageRef.ManifestKey == "" {
			continue
		}
		if _, ok := manifests[pageRef.ManifestKey]; ok {
			continue
		}

		report.BrokenPageCacheKeys = append(report.BrokenPageCacheKeys, pageRef.CacheKey)
		report.WouldDeletePageCacheKeys = append(report.WouldDeletePageCacheKeys, pageRef.CacheKey)
	}

	for key := range pageKeys {
		if manifest, ok := manifests[key]; ok {
			report.ReferencedManifestSize += manifest.Size
		}
	}

	for key := range manifests {
		if _, ok := pageKeys[key]; ok {
			continue
		}

		report.WouldDeleteManifestKeys = append(report.WouldDeleteManifestKeys, key)

		deps.CtxdLogger().Info(ctx, "album sprite: sprite cleanup candidate manifest",
			"manifest_key", key,
			"owner_count", len(manifests[key].Manifest.Albums),
			"owners", manifests[key].Manifest.Albums,
			"referenced_by_page_cache", false)
	}

	survivingManifestChunks := make(map[string]struct{}, len(manifests))
	for key, manifest := range manifests {
		if slicesContains(report.WouldDeleteManifestKeys, key) {
			continue
		}

		for chunk := range manifestChunks(manifest.Manifest) {
			survivingManifestChunks[chunk] = struct{}{}
		}
	}

	blobInfos, err := deps.AlbumSprites().StoredBlobs()
	if err != nil {
		return report, err
	}

	for _, blobInfo := range blobInfos {
		report.TotalBlobCount++
		report.TotalBlobSize += blobInfo.Size

		if _, ok := referencedChunks[blobInfo.Key]; ok {
			report.ReferencedBlobCount++
			report.ReferencedBlobSize += blobInfo.Size
		}

		if _, ok := survivingManifestChunks[blobInfo.Key]; !ok {
			report.WouldDeleteBlobKeys = append(report.WouldDeleteBlobKeys, blobInfo.Key)
			report.WouldDeleteBlobCount++
			report.WouldDeleteBlobSize += blobInfo.Size
			deps.CtxdLogger().Info(ctx, "album sprite: sprite cleanup candidate blob",
				"blob_key", blobInfo.Key,
				"blob_size", blobInfo.Size,
				"referenced_by_manifest_count", len(manifestChunkRefs[blobInfo.Key]),
				"referenced_by_manifest_keys", manifestChunkRefs[blobInfo.Key])
		}
	}

	sort.Strings(report.MissingManifestReferences)
	sort.Strings(report.BrokenPageCacheKeys)
	sort.Strings(report.WouldDeleteManifestKeys)
	sort.Strings(report.WouldDeletePageCacheKeys)
	sort.Strings(report.WouldDeleteBlobKeys)

	report.BrokenPageCacheCount = len(report.BrokenPageCacheKeys)
	report.WouldDeletePageCacheCount = len(report.WouldDeletePageCacheKeys)
	report.WouldDeleteManifestCount = len(report.WouldDeleteManifestKeys)

	if dryRun {
		return report, nil
	}

	for _, key := range report.WouldDeletePageCacheKeys {
		if err := deleteAlbumPageCache(ctx, deps.PersistentCacheStorage(), key); err != nil {
			return report, err
		}

		report.DeletedPageCacheKeys = append(report.DeletedPageCacheKeys, key)
	}

	report.DeletedPageCacheCount = len(report.DeletedPageCacheKeys)

	for _, key := range report.WouldDeleteManifestKeys {
		if err := deps.AlbumSprites().DeleteManifestRecord(ctx, key); err != nil {
			return report, err
		}

		report.DeletedManifestKeys = append(report.DeletedManifestKeys, key)
	}

	report.DeletedManifestCount = len(report.DeletedManifestKeys)

	wouldDeleteBlobSizes := make(map[string]int64, len(blobInfos))
	for _, blobInfo := range blobInfos {
		wouldDeleteBlobSizes[blobInfo.Key] = blobInfo.Size
	}

	for _, key := range report.WouldDeleteBlobKeys {
		if err := deps.AlbumSprites().DeleteBlob(ctx, key); err != nil {
			return report, err
		}

		report.DeletedBlobKeys = append(report.DeletedBlobKeys, key)
		report.DeletedBlobSize += wouldDeleteBlobSizes[key]
	}

	report.DeletedBlobCount = len(report.DeletedBlobKeys)
	report.WouldDeletePageCacheKeys = nil
	report.WouldDeletePageCacheCount = 0
	report.WouldDeleteManifestKeys = nil
	report.WouldDeleteManifestCount = 0
	report.WouldDeleteBlobKeys = nil
	report.WouldDeleteBlobCount = 0
	report.WouldDeleteBlobSize = 0

	return report, nil
}

func cachedAlbumPageManifestKeys(ctx context.Context, st *sqluct.Storage, sprites *sprite.Service) ([]cachedAlbumPageRef, map[string]struct{}, int, int, error) {
	rows, err := st.DB().DB.QueryContext(ctx, `SELECT key, val FROM record WHERE key LIKE 'album-page:page:%'`)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("query album page cache records: %w", err)
	}
	defer rows.Close()

	pageRefs := make([]cachedAlbumPageRef, 0)
	manifestKeys := make(map[string]struct{})
	pageCount := 0
	pageKeysWithSprites := 0

	for rows.Next() {
		var rec cachedJSONRecord
		if err := rows.Scan(&rec.Key, &rec.Val); err != nil {
			return nil, nil, 0, 0, fmt.Errorf("scan album page cache record: %w", err)
		}

		pageCount++

		var page cachedAlbumPageData
		if err := json.Unmarshal([]byte(rec.Val), &page); err != nil {
			return nil, nil, 0, 0, fmt.Errorf("decode album page cache %s: %w", rec.Key, err)
		}

		pageRef := cachedAlbumPageRef{
			CacheKey:    rec.Key,
			ManifestKey: page.SpriteManifestKey,
		}
		if len(page.SpriteSheets) == 0 {
			if pageRef.ManifestKey != "" {
				manifestKeys[pageRef.ManifestKey] = struct{}{}
			}
			pageRefs = append(pageRefs, pageRef)
			continue
		}

		pageKeysWithSprites++
		pageRef.HasSprites = true

		spriteImages := mergeCachedAlbumSpriteImages(&page)
		if len(spriteImages) == 0 {
			pageRefs = append(pageRefs, pageRef)
			continue
		}

		if pageRef.ManifestKey == "" {
			pageRef.ManifestKey = sprites.ManifestKey(spriteImages)
		}
		manifestKeys[pageRef.ManifestKey] = struct{}{}
		pageRefs = append(pageRefs, pageRef)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, 0, 0, fmt.Errorf("iterate album page cache records: %w", err)
	}

	return pageRefs, manifestKeys, pageCount, pageKeysWithSprites, nil
}

func storedSpriteManifests(ctx context.Context, st *sqluct.Storage) (map[string]storedManifestRecord, error) {
	rows, err := st.DB().DB.QueryContext(ctx, `SELECT key, val FROM record WHERE key LIKE 'album-sprite-manifest:%'`)
	if err != nil {
		return nil, fmt.Errorf("query sprite manifests: %w", err)
	}
	defer rows.Close()

	manifests := make(map[string]storedManifestRecord)

	for rows.Next() {
		var rec cachedJSONRecord
		if err := rows.Scan(&rec.Key, &rec.Val); err != nil {
			return nil, fmt.Errorf("scan sprite manifest: %w", err)
		}

		var manifest sprite.Manifest
		if err := json.Unmarshal([]byte(rec.Val), &manifest); err != nil {
			return nil, fmt.Errorf("decode sprite manifest %s: %w", rec.Key, err)
		}

		manifests[rec.Key] = storedManifestRecord{
			Manifest: manifest,
			Size:     int64(len(rec.Val)),
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sprite manifests: %w", err)
	}

	return manifests, nil
}

func mergeCachedAlbumSpriteImages(page *cachedAlbumPageData) []sprite.Image {
	seen := make(map[string]int)
	merged := make([]sprite.Image, 0)

	imageSets := make([][]cachedImage, 0, 1+len(page.SubAlbums))
	imageSets = append(imageSets, page.AlbumData.Images)
	for _, subAlbum := range page.SubAlbums {
		imageSets = append(imageSets, subAlbum.Images)
	}

	for _, set := range imageSets {
		for _, img := range set {
			if img.Is360Pano {
				continue
			}

			var h uniq.Hash
			if err := h.UnmarshalText([]byte(img.Hash)); err != nil {
				continue
			}

			item := sprite.Image{
				Hash:   h,
				Width:  img.Width,
				Height: img.Height,
				HasGPS: len(img.Gps) > 0 && string(img.Gps) != "null",
			}

			key := item.Hash.String()
			if idx, ok := seen[key]; ok {
				if item.HasGPS {
					merged[idx].HasGPS = true
				}

				continue
			}

			seen[key] = len(merged)
			merged = append(merged, item)
		}
	}

	return merged
}

func manifestChunks(manifest sprite.Manifest) map[string]struct{} {
	chunks := make(map[string]struct{})

	for _, item := range manifest.Images {
		if item.Chunk1x != "" {
			chunks[item.Chunk1x] = struct{}{}
		}
		if item.Chunk2x != "" {
			chunks[item.Chunk2x] = struct{}{}
		}
	}

	for _, item := range manifest.Markers {
		if item.Chunk1x != "" {
			chunks[item.Chunk1x] = struct{}{}
		}
		if item.Chunk2x != "" {
			chunks[item.Chunk2x] = struct{}{}
		}
	}

	return chunks
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

func allManifestChunkRefs(manifests map[string]storedManifestRecord) map[string][]string {
	refs := make(map[string][]string)

	for manifestKey, manifest := range manifests {
		for chunk := range manifestChunks(manifest.Manifest) {
			refs[chunk] = append(refs[chunk], manifestKey)
		}
	}

	for chunk := range refs {
		sort.Strings(refs[chunk])
	}

	return refs
}

func slicesContains(items []string, item string) bool {
	for _, v := range items {
		if v == item {
			return true
		}
	}

	return false
}

func deleteAlbumPageCache(ctx context.Context, st *sqluct.Storage, key string) error {
	res, err := st.DB().DB.ExecContext(ctx, `DELETE FROM record WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("delete album page cache %s: %w", key, err)
	}

	if _, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("album page cache rows affected %s: %w", key, err)
	}

	cacheKey := key
	const prefix = "album-page:"
	if len(cacheKey) > len(prefix) && cacheKey[:len(prefix)] == prefix {
		cacheKey = cacheKey[len(prefix):]
	}

	if _, err := st.DB().DB.ExecContext(ctx, `DELETE FROM cache_label WHERE cache_name = 'album-page' AND cache_key = ?`, cacheKey); err != nil {
		return fmt.Errorf("delete album page cache labels %s: %w", key, err)
	}

	return nil
}
