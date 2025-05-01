package cmd

import (
	"log/slog"
	"os"

	"context"

	sdhttp "github.com/SencilloDev/sencillo-go/transports/http"

	"github.com/SencilloDev/regoround/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startCmd = &cobra.Command{
	Use:          "start",
	Short:        "starts the service",
	RunE:         start,
	SilenceUsage: true,
}

func init() {
	// attach start subcommand to service subcommand
	serviceCmd.AddCommand(startCmd)
}

func start(cmd *cobra.Command, args []string) error {
	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}))

	ctx := context.Background()

	env := map[string]string{
		"token": "1234",
	}

	agent := service.NewAgent(service.AgentOpts{
		BundleName: "bundle.tar.gz",
		Env:        env,
		Logger:     logger,
	})

	agent.SetBundle(viper.GetString("bundle_path"))

	appCtx := service.AppContext{
		Agent: agent,
	}

	s := sdhttp.NewHTTPServer(
		sdhttp.SetServerPort(viper.GetInt("port")),
	)

	errChan := make(chan error, 1)

	s.RegisterSubRouter("/", service.MustGetRoutes())
	s.RegisterSubRouter("/api/v1", service.GetAPIRoutes(s.Logger, appCtx))

	go s.Serve(errChan)
	s.AutoHandleErrors(ctx, errChan)
	return nil
}
