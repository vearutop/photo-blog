package service

import (
	"github.com/bool64/brick"
	"github.com/bool64/ctxd"
	"github.com/swaggest/jsonform-go"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/files"
	"github.com/vearutop/photo-blog/internal/infra/geo/ors"
	"github.com/vearutop/photo-blog/internal/infra/image/cloudflare"
	"github.com/vearutop/photo-blog/internal/infra/image/faces"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	SchemaRepo           *jsonform.Repository
	AccessLogger         ctxd.Logger
	VisitorStatsInstance *visitor.Stats

	DepCacheInstance       *dep.Cache
	FilesProcessorInstance *files.Processor

	SettingsManagerInstance *settings.Manager

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

	PhotoMetaEnsurerProvider
	PhotoMetaFinderProvider

	PhotoGpxEnsurerProvider
	PhotoGpxFinderProvider

	TxtRendererProvider

	SiteVisitorEnsurerProvider
	SiteVisitorFinderProvider

	CommentMessageEnsurerProvider
	CommentMessageFinderProvider

	CommentThreadEnsurerProvider
	CommentThreadFinderProvider

	CloudflareImageClassifierInstance *cloudflare.ImageClassifier
	CloudflareImageDescriberInstance  *cloudflare.ImageDescriber

	FacesRecognizerInstance *faces.Recognizer
	ORS                     *ors.Client
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

func (l *Locator) CloudflareImageClassifier() *cloudflare.ImageClassifier {
	return l.CloudflareImageClassifierInstance
}

func (l *Locator) CloudflareImageDescriber() *cloudflare.ImageDescriber {
	return l.CloudflareImageDescriberInstance
}

func (l *Locator) FacesRecognizer() *faces.Recognizer {
	return l.FacesRecognizerInstance
}

func (l *Locator) OpenRouteService() *ors.Client {
	return l.ORS
}

func (l *Locator) VisitorStats() *visitor.Stats {
	return l.VisitorStatsInstance
}
