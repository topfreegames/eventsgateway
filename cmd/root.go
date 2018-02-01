package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var debug bool
var config *viper.Viper

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "TFGEventsGateway",
	Short: "TFG analytics events receiver",
	Long:  `TFG analytics events receiver`,
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug toggle")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "./config/local.yaml", "config file")
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	config = viper.New()
	if cfgFile != "" {
		config.SetConfigFile(cfgFile)
	}

	config.SetConfigType("yaml")
	config.SetEnvPrefix("eventsgateway")
	config.AddConfigPath(".")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	if err := config.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", config.ConfigFileUsed())
	}
}
