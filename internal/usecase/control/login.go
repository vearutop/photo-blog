package control

import (
	"context"

	"github.com/swaggest/usecase"
)

func Login() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, _ struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		_, _ = out.Write([]byte(`<html xmlns="http://www.w3.org/1999/xhtml">    
  <head><meta http-equiv="refresh" content="0;URL='/'" /></head>    
  <body><a href="/">Back</a></body>  
</html>`))

		return nil
	})

	return u
}
