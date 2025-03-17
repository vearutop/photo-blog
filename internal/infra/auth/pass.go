package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"math"
	"math/big"
	"strconv"

	"github.com/swaggest/jsonschema-go"
	"golang.org/x/crypto/argon2"
)

func Hash(in HashInput) string {
	key := argon2.Key([]byte(in.Pass), []byte(in.Salt), 3, 32*1024, 4, 32)
	return base64.RawStdEncoding.EncodeToString(key)
}

type (
	adminCtxKey struct{}
	botCtxKey   struct{}
)

func SetAdmin(ctx context.Context) context.Context {
	return context.WithValue(ctx, adminCtxKey{}, true)
}

func IsAdmin(ctx context.Context) bool {
	if ctx.Value(adminCtxKey{}) != nil {
		return true
	}

	return false
}

func SetBot(ctx context.Context) context.Context {
	return context.WithValue(ctx, botCtxKey{}, true)
}

func IsBot(ctx context.Context) bool {
	if ctx.Value(botCtxKey{}) != nil {
		return true
	}

	return false
}

type Salt string

type HashInput struct {
	Pass string `formData:"pass" format:"password"`
	Salt Salt   `formData:"salt"`
}

func (m Salt) PrepareJSONSchema(schema *jsonschema.Schema) error {
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt))
	if err != nil {
		return err
	}

	schema.WithDefault(strconv.FormatUint(n.Uint64(), 36))

	return nil
}
