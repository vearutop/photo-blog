package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/bool64/sqluct"
	"github.com/vearutop/photo-blog/internal/infra/storage/hashed"
)

const (
	// SettingsTable is the name of the table.
	SettingsTable = "settings"
)

func NewSettingsRepository(storage *sqluct.Storage) *SettingsRepository {
	return &SettingsRepository{
		st: storage,
		s:  sqluct.Table[settingsRow](storage, SettingsTable),
	}
}

type SettingsRepository struct {
	st *sqluct.Storage
	s  sqluct.StorageOf[settingsRow]
}

type settingsRow struct {
	Name  string `db:"name"`
	Value string `db:"value"`
}

func (r *SettingsRepository) Get(ctx context.Context, name string, value any) error {
	q := r.s.SelectStmt().Where(squirrel.Eq{r.s.Ref(&r.s.R.Name): name})

	s, err := r.s.Get(ctx, q)
	if err != nil {
		return hashed.AugmentErr(fmt.Errorf("get settings %q: %w", name, err))
	}

	if err := json.Unmarshal([]byte(s.Value), value); err != nil {
		return fmt.Errorf("unmarshal settings %s: %w", name, err)
	}

	return nil
}

func (r *SettingsRepository) Set(ctx context.Context, name string, value any) error {
	s := settingsRow{
		Name: name,
	}

	j, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal settings %s: %w", name, err)
	}

	s.Value = string(j)
	cond := squirrel.Eq{r.s.Ref(&r.s.R.Name): name}

	_, err = r.s.Get(ctx, r.s.SelectStmt().Where(cond))
	if err == nil {
		return hashed.AugmentReturnErr(r.s.UpdateStmt(s).Where(cond).ExecContext(ctx))
	}

	return hashed.AugmentReturnErr(r.s.InsertRow(ctx, s))
}
