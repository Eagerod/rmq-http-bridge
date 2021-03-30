package rmqhttp

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/rmqhttp/pkg/rmqhttp"
)

func mkDestroyCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "destroy",
		Short: "Destroy the delay infrastructure",
		RunE: func(cmd *cobra.Command, args []string) error {
			connectionString := getConnectionString()
			return rmqhttp.DestroyInfrastructure(connectionString)
		},
	}

	return cmd
}
