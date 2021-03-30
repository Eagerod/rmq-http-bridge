package rmqhttp

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/rmqhttp/pkg/rmqhttp"
)

func mkInitCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "init",
		Short: "Create the delay infrastructure",
		RunE: func(cmd *cobra.Command, args []string) error {
			connectionString := getConnectionString()
			return rmqhttp.DelayInfrastructure(connectionString)
		},
	}

	return cmd
}
