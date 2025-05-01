package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var cfg Config

var rootCmd = &cobra.Command{
	Use:   "regoroundctl",
	Short: "The app description",
}
var replacer = strings.NewReplacer("-", "_")

type Config struct {
	Port int `mapstructure:"port"`
}

func Execute() {
	viper.SetDefault("service-name", "regoround-local")
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.playground.json)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() {

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("json")
		viper.SetConfigName(".regoround")
	}

	viper.SetEnvPrefix("regoround")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(replacer)

	// If a config file is found, read it in.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := viper.ReadInConfig(); err == nil {
		logger.Debug(fmt.Sprintf("using config %s", viper.ConfigFileUsed()))
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		cobra.CheckErr(err)
	}
}
