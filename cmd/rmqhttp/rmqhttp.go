package rmqhttp

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
// It is empty, because the tool doesn't do anything without subcommands.
var rootCmd = &cobra.Command{
	Use:   "rmqhttp",
	Short: "proxy a RabbitMQ queue to HTTP endpoints",
	Long:  "Consume a RabbitMQ queue, and push the contents to HTTP endpoints.",
}

func Execute() {
	rootCmd.AddCommand(mkProduceCmd())
	rootCmd.AddCommand(mkConsumeCmd())
	rootCmd.AddCommand(mkInitCmd())
	rootCmd.AddCommand(mkDestroyCmd())

	log.SetLevel(log.DebugLevel)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
