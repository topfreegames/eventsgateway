package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/topfreegames/eventsgateway/app"
)

var host string
var port int

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts the events gateway",
	Long:  `starts the events gateway`,
	Run: func(cmd *cobra.Command, args []string) {
		log := logrus.New()
		if debug {
			log.SetLevel(logrus.DebugLevel)
		}
		if json {
			log.Formatter = new(logrus.JSONFormatter)
		}
		a, err := app.NewApp(host, port, log, config)
		if err != nil {
			log.Panic(err)
		}
		a.Run()
	},
}

func init() {
	startCmd.Flags().StringVarP(&host, "host", "b", "127.0.0.1", "the address of the interface to bind")
	startCmd.Flags().IntVarP(&port, "port", "p", 5000, "the port to bind")
	RootCmd.AddCommand(startCmd)
}
