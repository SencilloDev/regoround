//go:build js && wasm

package main

import (
	"bytes"
	"compress/flate"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"syscall/js"
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

func main() {
	js.Global().Set("decompressData", decompressWrapper())
	js.Global().Set("compressData", compressWrapper())
	fmt.Println("playground loaded")
	<-make(chan struct{})
}
