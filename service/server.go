// Copyright 2025 Sencillo
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
