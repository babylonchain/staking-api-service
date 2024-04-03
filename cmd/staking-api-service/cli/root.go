package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	defaultConfigFileName       = "config.yml"
	defaultGlobalParamsFileName = "global_params.json"
)

var (
	cfgPath          string
	globalParamsPath string
	rootCmd          = &cobra.Command{
		Use: "start-server",
	}
)

func Setup() error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	defaultConfigPath := getDefaultConfigFile(homePath, defaultConfigFileName)
	defaultGlobalParamsPath := getDefaultConfigFile(homePath, defaultGlobalParamsFileName)

	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", defaultConfigPath, fmt.Sprintf("config file (default %s)", defaultConfigPath))
	rootCmd.PersistentFlags().StringVar(&globalParamsPath, "params", defaultGlobalParamsPath, fmt.Sprintf("global params file (default %s)", defaultGlobalParamsPath))
	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func getDefaultConfigFile(homePath, filename string) string {
	return filepath.Join(homePath, filename)
}

func GetConfigPath() string {
	return cfgPath
}

func GetGlobalParamsPath() string {
	return globalParamsPath
}
