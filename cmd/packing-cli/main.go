// dubplate version: v1.0.0-2-g61f3327 (manually edited)

package main

import (
	"log"
	"os"
	"strings"

	"github.com/glynternet/pkg/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const appName = "packing-cli"

// to be changed using ldflags with the go build command
var version = "unknown"

func main() {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	out := os.Stdout

	cobra.OnInitialize(viperAutoEnvVar)

	var rootCmd = &cobra.Command{
		Use: appName,
	}

	rootCmd.AddCommand(cmd.NewVersionCmd(version, out))
	buildCmdTree(logger, out, rootCmd)

	err := viper.BindPFlags(rootCmd.Flags())
	if err != nil {
		logger.Printf("unable to BindPFlags: %v", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		logger.Println(err)
		os.Exit(1)
	}
}

func viperAutoEnvVar() {
	// TODO(glynternet): make this non-global
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match
}