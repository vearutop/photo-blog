package refresh

import "github.com/bool64/cache"

type Notifier struct {
	ii *cache.InvalidationIndex
}

func (n *Notifier) AlbumDependency() {

}

func (n *Notifier) AlbumChanged() {

}

func (n *Notifier) ServiceSettingsDependency() {

}

func (n *Notifier) ServiceSettingsChanged() {

}

func (n *Notifier) RefreshNotifier() *Notifier {
	return n
}
