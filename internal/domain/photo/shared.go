package photo

type Identity struct {
	ID int `db:"id,omitempty,serialIdentity" json:"id"`
}
