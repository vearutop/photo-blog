package short

import (
	"context"
	"math/rand"
	"strconv"

	"github.com/cespare/xxhash/v2"
)

type LinkFinder interface {
	FindByToken(ctx context.Context, token string) (*Link, error)
}

type LinkCreator interface {
	Create(link *Link) (*Link, error)
}

type Link struct {
	Token Token  `json:"token" db:"token"`
	URL   string `json:"url" db:"url"`
}

type Token string

const (
	tokenChars = "abcdefhijkmnpqrstuvwxyz23456789"
	tokenWidth = int64(len(tokenChars))
)

func MakeToken(longURL string, iteration int, length int) Token {
	s := strconv.Itoa(iteration) + ":" + longURL
	seed := xxhash.Sum64String(s)

	token := make([]byte, 0, tokenWidth)
	// token := ""
	r := rand.NewSource(int64(seed))
	for i := 0; i < length; i++ {
		c := r.Int63() % tokenWidth

		token = append(token, tokenChars[c])
		// token += string(tokenChars[c])
	}

	return Token(token)
}
