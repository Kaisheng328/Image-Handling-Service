package main

import (
	"Project/configs"
	"Project/routes"
	_ "image/png" // Import image packages to support PNG, JPEG, etc.
	"log"

	"github.com/gin-gonic/gin"
)

func main() {

	configs.InitiEnvConfigs()
	err := routes.InitializeClients()
	if err != nil {
		log.Fatalf("Failed to initialize clients: %v", err)
	}
	routes.InitializeRoutes()
	routes.Router.Static("/static", "./static")
	routes.Router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html") // Serve index.html directly
	})
	routes.Router.Run(":" + configs.EnvConfigs.LocalServerPort)

}
