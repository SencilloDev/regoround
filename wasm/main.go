//go:build js && wasm

package main

import (
	"bytes"
	"compress/flate"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"syscall/js"

	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage/inmem"
	"github.com/open-policy-agent/opa/util"
	//"github.com/tylermmorton/tmpl"
)

func compressWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return "Argument required"
		}
		data := args[0].String()
		b := base64.URLEncoding.EncodeToString([]byte(data))
		buff := bytes.NewBuffer([]byte(b))

		bout := &bytes.Buffer{}
		zwtr, err := flate.NewWriter(bout, flate.BestCompression)
		if err != nil {
			return err.Error()
		}
		_, err = io.Copy(zwtr, buff)
		if err != nil {
			return err.Error()
		}

		zwtr.Close()
		resp := base64.URLEncoding.EncodeToString(bout.Bytes())
		return resp
	})
}

func decompressWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return "Argument required"
		}
		data := args[0].String()
		old, err := base64.URLEncoding.DecodeString(data)
		if err != nil {
			return err.Error()
		}
		bin := bytes.NewBuffer(old)
		zrdr := flate.NewReader(bin)
		buf := bytes.Buffer{}
		_, err = io.Copy(&buf, zrdr)
		if err != nil {
			return err.Error()
		}
		l, err := base64.URLEncoding.DecodeString(buf.String())
		if err != nil {
			return err.Error()
		}
		return string(l)
	})
}

func evalRego(input, data, pkg string) (string, error) {
	var in any
	d := json.NewDecoder(bytes.NewBufferString(input))
	d.UseNumber()
	if err := d.Decode(&in); err != nil {
		return "", err
	}
	rg := rego.New(
		rego.Query("data.play"),
		rego.Module("play.rego", pkg),
		rego.Input(in),
	)
	if data != "" {
		var jdata map[string]any

		err := util.UnmarshalJSON([]byte(data), &jdata)
		if err != nil {
			return "", err
		}
		store := inmem.NewFromObject(jdata)
		f := rego.Store(store)
		f(rg)

	}

	ctx := context.Background()
	rs, err := rg.Eval(ctx)
	if err != nil {
		return "", err
	}
	if len(rs) == 0 {
		return "", err
	}

	m, err := json.Marshal(rs[0].Expressions[0].Value)
	if err != nil {
		return "", err
	}
	return string(m), nil
}

func regoWrapper() js.Func {
	jsonFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 3 {
			return "Invalid no arguments passed"
		}
		inputJSON := args[0].String()
		dataJSON := args[1].String()
		pkg := args[2].String()
		pretty, err := evalRego(inputJSON, dataJSON, pkg)
		if err != nil {
			fmt.Println(err)
			return err
		}
		return pretty
	})
	return jsonFunc
}

func main() {
	js.Global().Set("evalRego", regoWrapper())
	js.Global().Set("decompressData", decompressWrapper())
	js.Global().Set("compressData", compressWrapper())
	fmt.Println("playground loaded")
	<-make(chan struct{})
}
