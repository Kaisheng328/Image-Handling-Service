package routes

import (
	"Project/functions"
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

var (
	Router          = gin.Default()
	StorageClient   *storage.Client
	FirestoreClient *firestore.Client
	latestStatus    string = "API is working fine !!!!"
)

func InitializeRoutes() {
	publicRoutes := Router.Group("v1/")
	publicRoutes.GET("health", HealthCheck)
	publicRoutes.POST("health", HandlePost)
	publicRoutes.POST("health/water", HandleWatermarkImage)

}
func HealthCheck(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{
		"message": "API is working fine.",
	})
}

func HandlePost(c *gin.Context) {
	var requestBody struct {
		Base64Image string `json:"base64image"`
	}

	ctx := context.Background()
	sa := option.WithCredentialsFile("halogen-device-438608-v9-firebase-adminsdk-kwtb8-780d822bbb.json")
	StorageClient, err := storage.NewClient(ctx, sa)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Credential File"})
		fmt.Printf("error initializing app: %v\n", err)
		return
	}
	defer StorageClient.Close()
	// Initialize Firebase Storage
	FirestoreClient, err := firestore.NewClient(ctx, "halogen-device-438608-v9", sa, option.WithEndpoint("asia-southeast1-firestore.googleapis.com:443"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error initializing Firestore Client"})
		fmt.Printf("Error initializing Firestore client: %v\n", err)
		return
	}
	defer FirestoreClient.Close()

	// Parse the request body
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Println("Invalid Request Body")
		return
	}

	// Call the function to upload the image
	err = functions.UploadImageHandler(requestBody.Base64Image, StorageClient, FirestoreClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	currentTime := time.Now()
	timestamp := currentTime.Format("20060102_150405")
	latestStatus = fmt.Sprintf("image_%v uploaded successfully", timestamp)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}
func HandleWatermarkImage(c *gin.Context) {
	var requestBody struct {
		ImageID string `json:"imageID"` // Expecting the Image ID to be sent in the POST request
	}

	ctx := context.Background()
	sa := option.WithCredentialsFile("halogen-device-438608-v9-firebase-adminsdk-kwtb8-780d822bbb.json")
	StorageClient, err := storage.NewClient(ctx, sa)
	if err != nil {
		fmt.Printf("error initializing app: %v\n", err)
		return
	}
	defer StorageClient.Close()
	// Initialize Firebase Storage
	FirestoreClient, err := firestore.NewClient(ctx, "halogen-device-438608-v9", sa, option.WithEndpoint("asia-southeast1-firestore.googleapis.com:443"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error initializing Firestore Client"})
		fmt.Printf("Error initializing Firestore client: %v\n", err)
		return
	}
	defer FirestoreClient.Close()

	// Bind JSON request to requestBody struct
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err = functions.ProcessImageWithWatermark(requestBody.ImageID, StorageClient, FirestoreClient)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to Process"})
		return
	}
	latestStatus = fmt.Sprintf("watermarked_%s saved successfully", requestBody.ImageID)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}
