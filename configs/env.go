package configs

import (
	"log"

	"github.com/spf13/viper"
)

var EnvConfigs *envConfigs

type envConfigs struct {
	LocalServerPort string `mapstructure:"LOCAL_SERVER_PORT"`
	SecretKey       string `mapstructure:"SECRET_KEY"`
}

func InitiEnvConfigs() {
	EnvConfigs = loadEnvVariables()
}
func loadEnvVariables() (config *envConfigs) {
	viper.AddConfigPath(".")
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	if err := viper.ReadInConfig(); err != nil {
		log.Print("Error reading env file", err)
	}
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatal(err)
	}
	return
}
