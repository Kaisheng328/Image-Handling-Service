package main

import (
	"Project/configs"
	"Project/routes"
	_ "image/png" // Import image packages to support PNG, JPEG, etc.
)

func main() {

	configs.InitiEnvConfigs()
	routes.InitializeRoutes()
	routes.Router.Run(":" + configs.EnvConfigs.LocalServerPort)
	// r := gin.Default()

	// // Pass clients to the route handlers
	// r.POST("/v1/health", func(c *gin.Context) {
	// 	routes.HandlePost(c)
	// })

	// r.POST("/v1/health/Hi", func(c *gin.Context) {
	// 	routes.HandleWatermarkImage(c, StorageClient, FirestoreClient)
	// })

	// // Start the server
	// r.Run(":5000")
}
