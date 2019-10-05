package cmd

import (
	"log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// MustBindPFlags binds pflags for the given command and Fatals if there is an error
func MustBindPFlags(logger *log.Logger, cmd *cobra.Command) {
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		logger.Fatal(errors.Wrap(err, "binding pflags"))
	}
}
