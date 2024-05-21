package settings

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/swaggest/form/v5"
	"github.com/swaggest/refl"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/dep"
	"github.com/vearutop/photo-blog/internal/infra/image/cloudflare"
)

type Repository interface {
	Get(ctx context.Context, name string, value any) error
	Set(ctx context.Context, name string, value any) error
}

type Manager struct {
	r  Repository
	dc *dep.Cache

	mu          sync.Mutex
	security    Security
	appearance  Appearance
	maps        Maps
	visitors    Visitors
	storage     Storage
	privacy     Privacy
	externalAPI ExternalAPI
}

type Values interface {
	Security() Security
	Appearance() Appearance
	Maps() Maps
	Visitors() Visitors
	Storage() Storage
	Privacy() Privacy

	ExternalAPI() ExternalAPI
	CFImageClassifier() cloudflare.ImageWorkerConfig
	CFImageDescriber() cloudflare.ImageWorkerConfig
}

func NewManager(r Repository, dc *dep.Cache) (*Manager, error) {
	m := Manager{r: r, dc: dc}
	ctx := context.Background()

	return &m, errs(
		m.get(ctx, "security", &m.security),
		m.get(ctx, "appearance", &m.appearance),
		m.appearance.change(),
		m.get(ctx, "maps", &m.maps),
		m.get(ctx, "visitors", &m.visitors),
		m.get(ctx, "storage", &m.storage),
		m.get(ctx, "privacy", &m.privacy),
		m.get(ctx, "external_api", &m.externalAPI),
	)
}

func (m *Manager) get(ctx context.Context, name string, value any) error {
	if err := m.r.Get(ctx, name, value); errors.Is(err, status.NotFound) {
		return applyDefaults(value)
	} else {
		return err
	}
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

func applyDefaults(input any) error {
	defaults := url.Values{}
	defaultValDecoder := form.NewDecoder()
	defaultValDecoder.SetNamespacePrefix("[")
	defaultValDecoder.SetNamespaceSuffix("]")
	defaultValDecoder.RegisterTagNameFunc(func(field reflect.StructField) string {
		return field.Name
	})

	refl.WalkFieldsRecursively(reflect.ValueOf(input), func(v reflect.Value, sf reflect.StructField, path []reflect.StructField) {
		var key string

		for _, p := range path {
			if p.Anonymous {
				continue
			}

			if key == "" {
				key = p.Name
			} else {
				key += "[" + p.Name + "]"
			}
		}

		if key == "" {
			key = sf.Name
		} else {
			key += "[" + sf.Name + "]"
		}

		if d, ok := sf.Tag.Lookup("default"); ok {
			defaults[key] = []string{d}
		}
	})

	if len(defaults) == 0 {
		return nil
	}

	dec := defaultValDecoder

	return dec.Decode(input, defaults)
}
