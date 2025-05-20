package service

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"
)

func newLogger() *slog.Logger {
	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}))
	return logger
}
func TestEval(t *testing.T) {

	opts := AgentOpts{
		Logger: newLogger(),
		Env:    map[string]string{"": ""},
	}
	agent := NewAgent(opts)
	agent.SetBundle("")

	tt := []struct {
		name     string
		agent    *Agent
		input    []byte
		data     string
		pkg      string
		err      error
		expected []byte
	}{
		{
			name:  "basic",
			agent: agent,
			input: []byte(`{"name": "test"}`),
			pkg: `package play
			default allow := false
			allow if {
				input.name == "test"
			}`,
			data: "",
			expected: []byte(`{
	"allow": true
}`),
		},
		{
			name:  "data",
			agent: agent,
			input: []byte(`{"name": "test"}`),
			pkg: `package play
			default allow := false
			allow if {
				input.name == data.name
			}`,
			data: `{"name": "test"}`,
			expected: []byte(`{
	"allow": true
}`),
		},
	}

	for _, v := range tt {
		ctx := context.Background()
		t.Run(v.name, func(t *testing.T) {
			resp, err := v.agent.Eval(ctx, v.input, v.data, v.pkg)
			if err != nil && v.err == nil {
				t.Fatal(err)
			}
			if !bytes.Equal(v.expected, resp) {
				t.Errorf("expected\n%s\nbut got\n%s\n", string(v.expected), string(resp))
			}
		})
	}
}
