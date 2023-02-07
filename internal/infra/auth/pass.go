package auth

import (
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

type salt string

type HashInput struct {
	Pass string `formData:"pass" format:"password"`
	Salt salt   `formData:"salt"`
}

func (m salt) PrepareJSONSchema(schema *jsonschema.Schema) error {
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt))
	if err != nil {
		return err
	}

	schema.WithDefault(strconv.FormatUint(n.Uint64(), 36))

	return nil
}
