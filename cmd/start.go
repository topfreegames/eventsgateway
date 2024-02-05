// MIT License
//
// Copyright (c) 2018 Top Free Games
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cmd

import (
	"log"

	"github.com/mmcloughlin/professor"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/topfreegames/eventsgateway/v4/app"
	logruswrapper "github.com/topfreegames/eventsgateway/v4/logger/logrus"
	"github.com/topfreegames/eventsgateway/v4/metrics"
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
		a, err := app.NewApp(host, port, logruswrapper.NewWithLogger(log), config)
		if err != nil {
			log.Panic(err)
		}
		launchPProf()
		launchMetricsServer()
		a.Run()
	},
}

func launchPProf() {
	log.Println("Starting PProf HTTP server")
	config.SetDefault("pprof.enabled", true)
	config.SetDefault("pprof.address", "localhost:6060")
	if config.GetBool("pprof.enabled") {
		professor.Launch(config.GetString("pprof.address"))
	}
}

func launchMetricsServer() {
	config.SetDefault("prometheus.port", ":9091")
	httpPort := config.GetString("prometheus.port")
	log.Printf("Starting Metrics HTTP server at %s\n", httpPort)
	metrics.StartServer(httpPort)
}

func init() {
	startCmd.Flags().StringVarP(&host, "host", "b", "0.0.0.0", "the address of the interface to bind")
	startCmd.Flags().IntVarP(&port, "port", "p", 5000, "the port to bind")
	RootCmd.AddCommand(startCmd)
}
