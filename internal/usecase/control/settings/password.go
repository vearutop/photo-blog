package settings

import (
	"context"
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"strconv"

	"github.com/bool64/ctxd"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/vearutop/photo-blog/internal/infra/auth"
	"github.com/vearutop/photo-blog/internal/infra/settings"
)

type setSettingsDeps interface {
	CtxdLogger() ctxd.Logger
	SettingsManager() *settings.Manager
}

func SetPassword(deps setSettingsDeps) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input adminPass, output *struct{}) error {
		if input.Password != input.RepeatPassword {
			return status.Wrap(errors.New("passwords do not match"), status.InvalidArgument)
		}

		// Remove password.
		if input.Password == "" {
			return deps.SettingsManager().SetSecurity(ctx, settings.Security{})
		}

		n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt))
		if err != nil {
			return err
		}

		hi := auth.HashInput{
			Pass: input.Password,
			Salt: auth.Salt(strconv.FormatUint(n.Uint64(), 36)),
		}

		h := auth.Hash(hi)

		return deps.SettingsManager().SetSecurity(ctx, settings.Security{
			PassHash: h,
			PassSalt: string(hi.Salt),
		})
	})

	return u
}
