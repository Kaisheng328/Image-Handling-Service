package main

import (
	"Project/configs"
	"Project/routes"
	_ "image/png" // Import image packages to support PNG, JPEG, etc.
	"log"
)

func main() {
	configs.InitiEnvConfigs()
	err := routes.InitializeClients()
	if err != nil {
		log.Fatalf("Failed to initialize clients: %v", err)
	}
	routes.InitializeRoutes()
	routes.Router.Run(":" + configs.EnvConfigs.LocalServerPort)
}
