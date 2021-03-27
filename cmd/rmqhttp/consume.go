package rmqhttp

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/rmqhttp/pkg/rmqhttp"
)

func mkConsumeCmd() *cobra.Command {
	var consumeCmd = &cobra.Command{
		Use:   "worker",
		Short: "Pulls items off a RMQ queue, and sends them to their HTTP destination.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rmqhttp.ConsumeQueue("test")
			return nil
		},
	}

	return consumeCmd
}
