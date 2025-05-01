package service

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"

	sderrors "github.com/SencilloDev/sencillo-go/errors"
	sdhttp "github.com/SencilloDev/sencillo-go/transports/http"
)

//go:embed static/*
var static embed.FS

type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, a AppContext) error

type AppContext struct {
	Agent *Agent
}

func MustGetRoutes() []sdhttp.Route {
	content, err := fs.Sub(static, "static")
	if err != nil {
		log.Fatal(err)
	}

	return []sdhttp.Route{
		{
			Method:  http.MethodGet,
			Path:    "/",
			Handler: http.FileServer(http.FS(content)),
		},
	}
}

func GetAPIRoutes(l *slog.Logger, a AppContext) []sdhttp.Route {
	return []sdhttp.Route{
		{
			Method:  http.MethodPost,
			Path:    "/evaluate",
			Handler: ErrorWrapper(l, a, evaluate),
		},
	}
}

func ErrorWrapper(s *slog.Logger, a AppContext, f ErrorHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r, a)
		if err == nil {
			return
		}

		var ce sderrors.ClientError
		if errors.As(err, &ce) {
			w.WriteHeader(ce.Status)
			w.Write([]byte(ce.Body()))
			return
		}

		s.Error(fmt.Sprintf("status=%d, err=%v", http.StatusInternalServerError, err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(sdhttp.ErrInternalError.Error()))

	}
}
