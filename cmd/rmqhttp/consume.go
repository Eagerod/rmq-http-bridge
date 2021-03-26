package rmqhttp

import (
	// "errors"
	// "fmt"
)

import (
	// log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/rmqhttp/pkg/rmqhttp"
)

func mkConsumeCmd() *cobra.Command {
	// var consumeCmdTagSlice *[]string

	var consumeCmd = &cobra.Command{
		Use:   "worker",
		Short: "Pulls items off a RMQ queue, and sends them to their HTTP destination.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rmqhttp.ConsumeQueue("test")
			return nil
		},
	}

	// consumeCmdTagSlice = removeCmd.Flags().StringArrayP("tag", "t", []string{}, "remove resources with this tag")

	return consumeCmd
}
