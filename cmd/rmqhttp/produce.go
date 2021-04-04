package rmqhttp

import (
	"fmt"
	"net/http"
	"os"
)

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

import (
	"github.com/Eagerod/rmqhttp/pkg/rmqhttp"
)

func mkProduceCmd() *cobra.Command {
	var queueName string

	var cmd = &cobra.Command{
		Use:   "server",
		Short: "Receives HTTP POSTs on / and sends them to a queue.",
		RunE: func(cmd *cobra.Command, args []string) error {
			port := os.Getenv("PORT")
			if port == "" {
				port = "8080"
			}
			log.Infof("Starting RMQ HTTP Bridge on port %s", port)

			if queueName == "" {
				return fmt.Errorf("Must provide a queue to target.")
			}

			bindInterface := fmt.Sprintf("0.0.0.0:%s", port)

			connectionString := getConnectionString()
			r := mux.NewRouter()

			hc := rmqhttp.NewHttpController()
			hc.Connect(connectionString, queueName)

			r.HandleFunc("/", hc.HttpHandler).Methods("POST")
			r.HandleFunc("/health", hc.HealthHandler).Methods("GET")
			http.Handle("/", r)
			return http.ListenAndServe(bindInterface, nil)
		},
	}

	cmd.Flags().StringVarP(&queueName, "queue", "q", "", "Queue to write to")

	return cmd
}
