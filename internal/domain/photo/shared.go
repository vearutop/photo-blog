package photo

import (
	"time"
)

type Identity struct {
	ID int `db:"id,omitempty,serialIdentity" json:"id"`
}

type Time struct {
	CreatedAt time.Time `db:"created_at,omitempty" json:"created_at"`
}

type HashHead struct {
	Time
	Hash Hash `db:"hash" json:"hash" description:"image hash"`
}

func (h *HashHead) HashPtr() *Hash {
	return &h.Hash
}

func (h *HashHead) SetCreatedAt(t time.Time) {
	h.CreatedAt = t
}
