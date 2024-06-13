// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2017 Top Free Games <backend@tfgco.com>

package testing

import (
	"github.com/spf13/viper"
	"strings"
)

// GetDefaultConfig returns the configuration at ./config/test.yaml
func GetDefaultConfig() (*viper.Viper, error) {
	cfg := viper.New()
	cfg.SetConfigName("test")
	cfg.SetConfigType("yaml")
	cfg.SetEnvPrefix("eventsgateway")
	cfg.AddConfigPath(".")
	cfg.AddConfigPath("./config")
	cfg.AddConfigPath("../config")
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cfg.AutomaticEnv()

	// If a config file is found, read it in.
	if err := cfg.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg.Set("prometheus.enabled", "false")

	return cfg, nil
}
