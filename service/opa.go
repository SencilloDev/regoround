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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	serrors "github.com/SencilloDev/sencillo-go/errors"
	"github.com/nats-io/nats.go"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/cover"
	"github.com/open-policy-agent/opa/v1/metrics"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/topdown/cache"
)

var (
	ErrNotFound error = errors.New("package not found")
)

type BundleModifyFunc func(b bundle.Bundle) (bundle.Bundle, error)

type Agent struct {
	BundleName  string
	RawBundle   bundle.Bundle
	ObjectStore nats.ObjectStore
	OPAStore    storage.Store
	mutex       sync.RWMutex
	Logger      *slog.Logger
	Env         map[string]string
	astFunc     func(*rego.Rego)
	Compiler    *ast.Compiler
	Modifiers   []BundleModifyFunc
	Cache       cache.InterQueryCache
}

type AgentOpts struct {
	BundleName string
	Logger     *slog.Logger
	Env        map[string]string
	Modifiers  []BundleModifyFunc
}

func NewAgent(opts AgentOpts) *Agent {
	config, _ := cache.ParseCachingConfig(nil)
	interQueryCache := cache.NewInterQueryCache(config)
	a := &Agent{
		BundleName: opts.BundleName,
		Logger:     opts.Logger,
		Env:        opts.Env,
		OPAStore:   inmem.New(),
		Compiler:   ast.NewCompiler(),
		Modifiers:  opts.Modifiers,
		Cache:      interQueryCache,
	}
	if opts.Env != nil {
		a.SetRuntime()
	}

	return a
}

func (a *Agent) SetRuntime() {
	obj := ast.NewObject()
	env := ast.NewObject()
	for k, v := range a.Env {
		env.Insert(ast.StringTerm(k), ast.StringTerm(v))
	}
	obj.Insert(ast.StringTerm("env"), ast.NewTerm(env))
	a.astFunc = rego.Runtime(obj.Get(ast.StringTerm("env")))
}

func (a *Agent) SetBundle(path string) error {
	if path == "" {
		b := bundle.Bundle{}
		b.Data = make(map[string]any)
		a.RawBundle = b
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}

	// build new reader from tarball retrieved over NATS
	tarball := bundle.NewCustomReader(bundle.NewTarballLoaderWithBaseURL(f, ""))
	b, err := tarball.Read()
	if err != nil {
		return fmt.Errorf("error reading bundle: %v", err)
	}
	a.Logger.Info("generated tarball from bundle successfully")

	a.RawBundle = b

	return nil
}

func deepMerge(dst, src map[string]any) map[string]any {
	for k, v := range src {
		if vMap, ok := v.(map[string]any); ok {
			if dstVMap, found := dst[k].(map[string]any); found {
				dst[k] = deepMerge(dstVMap, vMap)
			} else {
				dst[k] = deepMerge(make(map[string]any), vMap)
			}
		} else {
			dst[k] = v
		}
	}
	return dst
}

func (a *Agent) GetStorage(ctx context.Context, data map[string]any) (storage.Store, *ast.Compiler, error) {
	store := inmem.New()
	rawBundle := a.RawBundle.Copy()

	rawBundle.Data = deepMerge(rawBundle.Data, data)

	bundles := map[string]*bundle.Bundle{
		"playground": &rawBundle,
	}
	c := storage.NewContext()
	txn, err := store.NewTransaction(ctx, storage.TransactionParams{Context: c, Write: true})
	if err != nil {
		return nil, nil, err
	}
	compiler := ast.NewCompiler()
	opts := bundle.ActivateOpts{
		Ctx:      ctx,
		Store:    store,
		Bundles:  bundles,
		Txn:      txn,
		TxnCtx:   c,
		Compiler: compiler,
		Metrics:  metrics.New(),
	}

	if err := bundle.Activate(&opts); err != nil {
		a.Logger.Error(err.Error())
		return nil, nil, err
	}

	if err := store.Commit(ctx, txn); err != nil {
		return nil, nil, err
	}

	return store, compiler, nil
}

// Eval evaluates the input against the policy package
func (a *Agent) Eval(ctx context.Context, input []byte, reqData, pkg string) (Response, error) {
	if input == nil {
		return Response{}, serrors.NewClientError(fmt.Errorf("input required"), 422)
	}

	if pkg == "" {
		return Response{}, serrors.NewClientError(fmt.Errorf("package name required"), 422)
	}

	a.Logger.Debug(fmt.Sprintf("evaluating package: %s", pkg))
	a.Logger.Debug(fmt.Sprintf("parsing input: %v", string(input)))
	data, _, err := readInputGetV1(input)
	if err != nil {
		return Response{}, serrors.NewClientError(err, 400)
	}

	if reqData == "" {
		reqData = `{}`
	}

	var tempData map[string]any
	if err := json.Unmarshal([]byte(reqData), &tempData); err != nil {
		return Response{}, serrors.NewClientError(err, 400)
	}

	store, compiler, err := a.GetStorage(ctx, tempData)
	if err != nil {
		return Response{}, serrors.NewClientError(err, 400)
	}

	c := storage.NewContext()
	txn, err := store.NewTransaction(ctx, storage.TransactionParams{Context: c, Write: true})
	if err != nil {
		return Response{}, serrors.NewClientError(err, 400)
	}

	cov := cover.New()
	parsed, err := ast.ParseModule("play.rego", pkg)
	if err != nil {
		return Response{}, serrors.NewClientError(err, 400)
	}

	r := rego.New(
		rego.Compiler(compiler),
		rego.Query("data.play"),
		rego.ParsedModule(parsed),
		rego.Transaction(txn),
		rego.Store(store),
		rego.ParsedInput(data),
		rego.InterQueryBuiltinCache(a.Cache),
		rego.QueryTracer(cov),
		a.astFunc,
	)

	prepared, err := r.PrepareForEval(ctx)
	if err != nil {
		return Response{}, serrors.NewClientError(err, 400)
	}

	results, err := prepared.Eval(ctx,
		rego.EvalParsedInput(data),
		rego.EvalTransaction(txn),
		rego.EvalInterQueryBuiltinCache(a.Cache),
		rego.EvalTracer(cov),
	)
	if err != nil {
		return Response{}, serrors.NewClientError(err, 400)
	}

	report := cov.Report(map[string]*ast.Module{
		"play.rego": parsed,
	})

	fr, ok := report.Files["play.rego"]
	if !ok {
		return Response{}, fmt.Errorf("error getting files from report")
	}

	if len(results) < 1 {
		return Response{}, ErrNotFound
	}

	value, err := json.MarshalIndent(results[0].Expressions[0].Value, "", "	")
	if err != nil {
		return Response{}, err
	}

	a.Logger.Debug(fmt.Sprintf("response: %s", string(value)))

	return Response{
		Data:     value,
		Coverage: fr,
	}, nil

}

func readInputGetV1(data []byte) (ast.Value, *any, error) {
	var input any
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid input: %w", err)
	}
	v, err := ast.InterfaceToValue(input)
	return v, &input, err
}

func CustomData(ctx context.Context, data []byte) (bundle.Bundle, error) {
	b := bundle.Bundle{}
	var bundleData map[string]any
	if err := json.Unmarshal(data, &bundleData); err != nil {
		return b, err
	}

	custData := map[string]any{
		"custom": bundleData,
	}
	b.Data["new"] = custData

	return b, nil
}
