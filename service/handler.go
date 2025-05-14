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
	"encoding/json"
	"net/http"

	sderrors "github.com/SencilloDev/sencillo-go/errors"
)

type Request struct {
	Input   string `json:"input"`
	Data    string `json:"data"`
	Package string `json:"package"`
}

func evaluate(w http.ResponseWriter, r *http.Request, a AppContext) error {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return sderrors.NewClientError(err, 400)
	}

	resp, err := a.Agent.Eval(r.Context(), []byte(req.Input), req.Data, req.Package)
	if err != nil {
		return err
	}

	w.Write(resp)

	return nil
}
