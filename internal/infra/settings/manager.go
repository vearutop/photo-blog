package settings

import (
	"context"
	"errors"
	"fmt"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"strings"
	"sync"

	"github.com/swaggest/usecase/status"
)

type Repository interface {
	Get(ctx context.Context, name string, value any) error
	Set(ctx context.Context, name string, value any) error
}

type Manager struct {
	r  Repository
	dc *dep.Cache

	mu         sync.Mutex
	security   Security
	appearance Appearance
	maps       Maps
	visitors   Visitors
	storage    Storage
}

type Values interface {
	Security() Security
	Appearance() Appearance
	Maps() Maps
	Visitors() Visitors
	Storage() Storage
}

func NewManager(r Repository, dc *dep.Cache) (*Manager, error) {
	m := Manager{r: r, dc: dc}
	ctx := context.Background()

	return &m, errs(
		m.r.Get(ctx, "security", &m.security),
		m.r.Get(ctx, "appearance", &m.appearance),
		m.appearance.change(),
		m.r.Get(ctx, "maps", &m.maps),
		m.r.Get(ctx, "visitors", &m.visitors),
		m.r.Get(ctx, "storage", &m.storage),
	)
}

func (m *Manager) set(ctx context.Context, name string, value any) error {
	if err := m.r.Set(ctx, name, value); err != nil {
		return err
	}

	if err := m.dc.ServiceSettingsChanged(ctx); err != nil {
		return fmt.Errorf("invalidate settings cache deps: %w", err)
	}

	return nil
}

func errs(es ...error) error {
	var s []string

	for _, e := range es {
		if e != nil && !errors.Is(e, status.NotFound) {
			s = append(s, e.Error())
		}
	}

	if len(s) == 0 {
		return nil
	}

	return errors.New(strings.Join(s, ", "))
}
