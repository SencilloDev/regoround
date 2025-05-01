package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/metrics"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/topdown/cache"
	"github.com/open-policy-agent/opa/v1/util"
)

var (
	ErrNotFound error = fmt.Errorf("package not found")
)

type BundleModifyFunc func(b bundle.Bundle) (bundle.Bundle, error)

type Agent struct {
	BundleName  string
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
		Cache:      cache.InterQueryCache(interQueryCache),
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

	for _, v := range a.Modifiers {
		a.Logger.Debug("modifying bundle")
		b, err = v(b)
		if err != nil {
			return fmt.Errorf("error in bundle modifier: %w", err)
		}
	}

	if err := a.Activate(ctx, b); err != nil {
		return err
	}
	a.Logger.Info("activated bundle successfully")

	return nil
}

func (a *Agent) Activate(ctx context.Context, b bundle.Bundle) error {
	bundles := map[string]*bundle.Bundle{
		"playground": &b,
	}
	c := storage.NewContext()
	txn, err := a.OPAStore.NewTransaction(ctx, storage.TransactionParams{Context: c, Write: true})
	if err != nil {
		return err
	}
	opts := bundle.ActivateOpts{
		Ctx:      ctx,
		Store:    a.OPAStore,
		Bundles:  bundles,
		Txn:      txn,
		TxnCtx:   c,
		Compiler: a.Compiler,
		Metrics:  metrics.New(),
	}

	if err := bundle.Activate(&opts); err != nil {
		a.Logger.Error(err.Error())
		a.OPAStore.Abort(ctx, txn)
		return err
	}

	return a.OPAStore.Commit(ctx, txn)
}

// Eval evaluates the input against the policy package
func (a *Agent) Eval(ctx context.Context, input []byte, reqData, pkg string) ([]byte, error) {
	if input == nil {
		return nil, fmt.Errorf("input required")
	}

	if pkg == "" {
		return nil, fmt.Errorf("package name required")
	}

	a.Logger.Debug(fmt.Sprintf("evaluating package: %s", pkg))
	a.Logger.Debug(fmt.Sprintf("parsing input: %v", string(input)))
	data, _, err := readInputGetV1(input)
	if err != nil {
		a.Logger.Error(err.Error())
		return nil, err
	}

	store := a.OPAStore

	a.mutex.RLock()
	defer a.mutex.RUnlock()
	c := storage.NewContext()
	txn, err := store.NewTransaction(ctx, storage.TransactionParams{Context: c, Write: true})
	if err != nil {
		a.Logger.Error(err.Error())
		return nil, err
	}
	defer store.Abort(ctx, txn)

	if reqData != "" {
		var jdata map[string]any

		if err := util.UnmarshalJSON([]byte(reqData), &jdata); err != nil {
			return nil, err
		}
		if err := store.Write(ctx, txn, storage.AddOp, storage.Path{}, jdata); err != nil {
			return nil, err
		}
	}

	r := rego.New(
		rego.Compiler(a.Compiler),
		rego.Query("data.play"),
		rego.Module("play.rego", pkg),
		rego.Transaction(txn),
		rego.Store(store),
		rego.ParsedInput(data),
		rego.InterQueryBuiltinCache(a.Cache),
		a.astFunc,
	)

	prepared, err := r.PrepareForEval(ctx)
	if err != nil {

		a.Logger.Error(err.Error())
		return []byte(err.Error()), nil
	}

	results, err := prepared.Eval(ctx,
		rego.EvalParsedInput(data),
		rego.EvalTransaction(txn),
		rego.EvalInterQueryBuiltinCache(a.Cache),
	)
	if err != nil {
		a.Logger.Error(err.Error())
		return []byte(err.Error()), nil
	}

	if len(results) < 1 {
		return nil, ErrNotFound
	}

	value, err := json.MarshalIndent(results[0].Expressions[0].Value, "", "	")
	if err != nil {
		return nil, err
	}

	a.Logger.Debug(fmt.Sprintf("response: %s", string(value)))

	return value, nil
}

func readInputGetV1(data []byte) (ast.Value, *any, error) {
	var input any
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid input: %w", err)
	}
	v, err := ast.InterfaceToValue(input)
	return v, &input, err
}
