package cmd

import (
	"git.topfreegames.com/eventsgateway/api"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
		a, err := api.NewApp(host, port, log)
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
