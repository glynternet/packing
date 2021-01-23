// Code generated by dubplate v0.10.1 DO NOT EDIT.
// Implement the following function to use this boilerplate
// func buildCmdTree(logger log.Logger, out io.Writer, rootCmd *cobra.Command) {}

package main

import (
	"os"
	"strings"

	"github.com/glynternet/pkg/cmd"
	"github.com/glynternet/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const appName = "packing-cli"

// to be changed using ldflags with the go build command
var version = "unknown"

func main() {
	logger := log.NewLogger(os.Stderr)
	out := os.Stdout

	cobra.OnInitialize(viperAutoEnvVar)

	var rootCmd = &cobra.Command{
		Use: appName,
	}

	rootCmd.AddCommand(cmd.NewVersionCmd(version, out))
	buildCmdTree(logger, out, rootCmd)

	err := viper.BindPFlags(rootCmd.Flags())
	if err != nil {
		_ = logger.Log(
			log.Message("Unable to BindPFlags"),
			log.ErrorMessage(err))
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		_ = logger.Log(
			log.Message("Error executing root command"),
			log.ErrorMessage(err))
		os.Exit(1)
	}
}

func viperAutoEnvVar() {
	// TODO(glynternet): make this non-global
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match
}
