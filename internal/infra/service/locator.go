package service

import (
	"github.com/bool64/brick"
	"github.com/bool64/cache"
	"github.com/bool64/ctxd"
	"github.com/swaggest/jsonform-go"
	"github.com/vearutop/dbcon/dbcon"
	"github.com/vearutop/image-prompt/multi"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/files"
	"github.com/vearutop/photo-blog/internal/infra/geo/ors"
	"github.com/vearutop/photo-blog/internal/infra/image/cloudflare"
	"github.com/vearutop/photo-blog/internal/infra/image/faces"
	"github.com/vearutop/photo-blog/internal/infra/settings"
	"github.com/vearutop/photo-blog/internal/infra/storage/visitor"
	"github.com/vearutop/photo-blog/pkg/qlite"
)

// Locator defines application resources.
type Locator struct {
	*brick.BaseLocator

	SchemaRepo            *jsonform.Repository
	AccessLogger          ctxd.Logger
	VisitorStatsInstance  *visitor.StatsRepository
	MapTilesCacheInstance *cache.FailoverOf[[]byte]

	DepCacheInstance       *dep.Cache
	FilesProcessorInstance *files.Processor

	SettingsManagerInstance *settings.Manager

	QueueBrokerInstance *qlite.Broker

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

	FavoriteRepositoryProvider

	CloudflareImageClassifierInstance *cloudflare.ImageClassifier
	CloudflareImageDescriberInstance  *cloudflare.ImageDescriber

	FacesRecognizerInstance *faces.Recognizer
	ORS                     *ors.Client

	ImagePrompterInstance *multi.ImagePrompter

	dbInstances []dbcon.DBInstance
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

func (l *Locator) ImagePrompter() *multi.ImagePrompter {
	return l.ImagePrompterInstance
}

func (l *Locator) OpenRouteService() *ors.Client {
	return l.ORS
}

func (l *Locator) VisitorStats() *visitor.StatsRepository {
	return l.VisitorStatsInstance
}

func (l *Locator) AddDBConInstance(db dbcon.DBInstance) {
	l.dbInstances = append(l.dbInstances, db)
}

func (l *Locator) DBInstances() []dbcon.DBInstance {
	return l.dbInstances
}

func (l *Locator) Prompter() dbcon.Prompter {
	return nil
}

func (l *Locator) QueueBroker() *qlite.Broker {
	return l.QueueBrokerInstance
}

func (l *Locator) MapTilesCache() *cache.FailoverOf[[]byte] {
	return l.MapTilesCacheInstance
}
