package service

import (
	"github.com/bool64/brick"
	"github.com/bool64/ctxd"
	"github.com/swaggest/jsonform-go"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/files"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	SchemaRepo   *jsonform.Repository
	AccessLogger ctxd.Logger

	DepCacheInstance       *dep.Cache
	FilesProcessorInstance *files.Processor

	SettingsManagerInstance *settings.Manager
	StorageStats            *storage.Stats

	Config Config

	PhotoAlbumEnsurerProvider
	PhotoAlbumUpdaterProvider
	PhotoAlbumDeleterProvider
	PhotoAlbumFinderProvider

	PhotoAlbumImageAdderProvider
	PhotoAlbumImageFinderProvider
	PhotoAlbumImageDeleterProvider

	PhotoImageEnsurerProvider
	PhotoImageUpdaterProvider
	PhotoImageFinderProvider
	PhotoImageIndexerProvider

	PhotoThumbnailerProvider

	PhotoExifEnsurerProvider
	PhotoExifFinderProvider

	PhotoGpsEnsurerProvider
	PhotoGpsFinderProvider

	PhotoGpxEnsurerProvider
	PhotoGpxFinderProvider

	TxtRendererProvider
}

// ServiceConfig gives access to service configuration.
func (l *Locator) ServiceConfig() Config {
	return l.Config
}

func (l *Locator) SchemaRepository() *jsonform.Repository {
	return l.SchemaRepo
}

func (l *Locator) AccessLog() ctxd.Logger {
	return l.AccessLogger
}

func (l *Locator) DepCache() *dep.Cache {
	return l.DepCacheInstance
}

func (l *Locator) FilesProcessor() *files.Processor {
	return l.FilesProcessorInstance
}

func (l *Locator) SettingsManager() *settings.Manager {
	return l.SettingsManagerInstance
}

func (l *Locator) Settings() settings.Values {
	return l.SettingsManagerInstance
}

func (l *Locator) Stats() *storage.Stats {
	return l.StorageStats
}
