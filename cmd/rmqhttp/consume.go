package rmqhttp

import (
	"fmt"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/rmqhttp/pkg/rmqhttp"
)

func mkConsumeCmd() *cobra.Command {
	var queueName string

	var cmd = &cobra.Command{
		Use:   "worker",
		Short: "Pulls items off a RMQ queue, and sends them to their HTTP destination.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if queueName == "" {
				return fmt.Errorf("Must provide queue name to consume")
			}

			rmqhttp.ConsumeQueue(queueName)
			return nil
		},
	}

	cmd.Flags().StringVarP(&queueName, "queue", "q", "", "Queue to consume")

	return cmd
}
