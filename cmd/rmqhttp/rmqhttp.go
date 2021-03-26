package rmqhttp

import (
	// "fmt"
	// "os"
	// "strings"
)

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	// "github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rmqhttp",
	Short: "proxy a RabbitMQ queue to HTTP endpoints",
	Long: "Consume a RabbitMQ queue, and push the contents to HTTP endpoints.",
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.AddCommand(mkProduceCmd())
	rootCmd.AddCommand(mkConsumeCmd())

	log.SetLevel(log.DebugLevel)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
