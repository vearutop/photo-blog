package storage

import (
	"database/sql"
	"errors"
	"github.com/swaggest/usecase/status"
	"modernc.org/sqlite"
)

func augmentErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return status.Wrap(err, status.NotFound)
	}

	var se *sqlite.Error

	if errors.As(err, &se) {
		if se.Code() == 2067 || se.Code() == 1555 {
			err = status.Wrap(err, status.AlreadyExists)
		}
	}

	return err
}

func augmentResErr[V any](res V, err error) (V, error) {
	return res, augmentErr(err)
}
