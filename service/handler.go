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
