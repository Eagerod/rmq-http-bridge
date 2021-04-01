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
	var consumers int

	var cmd = &cobra.Command{
		Use:   "worker",
		Short: "Pulls items off a RMQ queue, and sends them to their HTTP destination.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if queueName == "" {
				return fmt.Errorf("Must provide queue name to consume")
			}

			connectionString := getConnectionString()
			rmqhttp.ConsumeQueue(connectionString, queueName, consumers)
			return nil
		},
	}

	cmd.Flags().StringVarP(&queueName, "queue", "q", "", "Queue to consume")
	cmd.Flags().IntVarP(&consumers, "consumers", "c", 1, "Number of consumers to run")

	return cmd
}
