package cmd

import (
	"github.com/glynternet/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// MustBindPFlags binds pflags for the given command and Fatals if there is an error
func MustBindPFlags(logger log.Logger, cmd *cobra.Command) {
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		_ = logger.Log(
			log.Message("Error binding pflags"),
			log.Error(err),
		)
	}
}
