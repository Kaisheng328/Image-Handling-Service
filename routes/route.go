package routes

import (
	"Project/configs"
	"Project/functions"
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

var (
	Router          = gin.Default()
	StorageClient   *storage.Client
	FirestoreClient *firestore.Client
	latestStatus    string
)

func InitializeRoutes() {
	publicRoutes := Router.Group("v1/")
	publicRoutes.GET("health", HealthCheck)
	publicRoutes.GET("health/:id/small", GetSmallImagePath)
	publicRoutes.GET("health/:id/medium", GetMediumImagePath)
	publicRoutes.GET("health/:id/large", GetLargeImagePath)

	publicRoutes.POST("health", PostImage)
	publicRoutes.POST("health/small", PostImageSmall)
	publicRoutes.POST("health/medium", PostImageMedium)
	publicRoutes.POST("health/large", PostImageLarge)
	publicRoutes.POST("health/small/water", PostSmallWatermarkImage)
	publicRoutes.POST("health/medium/water", PostMediumWatermarkImage)
	publicRoutes.POST("health/large/water", PostLargeWatermarkImage)

}

func InitializeClients() error {

	ctx := context.Background()

	// Initialize Storage client
	var err error
	credsOption := option.WithCredentialsFile(configs.EnvConfigs.GoogleCred)
	StorageClient, err = storage.NewClient(ctx, credsOption)
	if err != nil {
		return fmt.Errorf("failed to initialize storage client: %v", err)
	}

	// Initialize Firestore client
	FirestoreClient, err = firestore.NewClient(ctx, "halogen-device-438608-v9", credsOption)
	if err != nil {
		return fmt.Errorf("failed to initialize Firestore client: %v", err)
	}

	return nil
}
func FetchCredentialsFromSecretManager(secretName string) ([]byte, error) {
	// Create the Secret Manager client
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager client: %v", err)
	}
	defer client.Close()

	// Build the request
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	// Access the secret version
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to access secret version: %v", err)
	}

	// Return the secret payload (credentials JSON)
	return result.Payload.Data, nil
}
func HealthCheck(context *gin.Context) {
	latestStatus = "API is working fine !!!!"
	context.JSON(http.StatusOK, gin.H{
		"message": latestStatus,
	})
}
func GetSmallImagePath(context *gin.Context) {
	ImageID := context.Param("id")
	smallPath, err := functions.GetSmallImageDetailFromFirestore(FirestoreClient, ImageID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	latestStatus = fmt.Sprintf("image_%v uploaded successfully", smallPath)
	context.JSON(http.StatusOK, gin.H{
		"smallPath": smallPath,
	})
}
func GetMediumImagePath(context *gin.Context) {
	ImageID := context.Param("id")
	mediumPath, err := functions.GetMediumImageDetailFromFirestore(FirestoreClient, ImageID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	latestStatus = fmt.Sprintf("image_%v uploaded successfully", mediumPath)
	context.JSON(http.StatusOK, gin.H{
		"mediumPath": mediumPath,
	})
}
func GetLargeImagePath(context *gin.Context) {
	ImageID := context.Param("id")
	largePath, err := functions.GetLargeImageDetailFromFirestore(FirestoreClient, ImageID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	latestStatus = fmt.Sprintf("image_%v uploaded successfully", largePath)
	context.JSON(http.StatusOK, gin.H{
		"largePath": largePath,
	})
}

func PostImage(c *gin.Context) {
	var requestBody struct {
		Base64Image string `json:"base64image"`
	}

	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Println("Invalid Request Body")
		return
	}
	currentTime := time.Now()
	timestamp := currentTime.Format("20060102_150405")

	// Call the function to upload the image
	err := functions.UploadImageHandler(requestBody.Base64Image, StorageClient, FirestoreClient, timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	latestStatus = fmt.Sprintf("image_%v uploaded successfully", timestamp)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}
func PostImageSmall(c *gin.Context) {
	var requestBody struct {
		ImageID string `json:"imageID"` // Expecting the Image ID to be sent in the POST request
	}
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	err := functions.ProcessResizeSmallImage(requestBody.ImageID, StorageClient, FirestoreClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Step 4: Respond with success status
	latestStatus := fmt.Sprintf("image_%v resized to small successfully", requestBody.ImageID)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}

func PostImageMedium(c *gin.Context) {
	var requestBody struct {
		ImageID string `json:"imageID"` // Expecting the Image ID to be sent in the POST request
	}
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	err := functions.ProcessResizeMediumImage(requestBody.ImageID, StorageClient, FirestoreClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Step 4: Respond with success status
	latestStatus := fmt.Sprintf("image_%v resized to medium successfully", requestBody.ImageID)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}
func PostImageLarge(c *gin.Context) {
	var requestBody struct {
		ImageID string `json:"imageID"` // Expecting the Image ID to be sent in the POST request
	}
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	err := functions.ProcessResizeLargeImage(requestBody.ImageID, StorageClient, FirestoreClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Step 4: Respond with success status
	latestStatus := fmt.Sprintf("image_%v resized to large successfully", requestBody.ImageID)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}
func PostSmallWatermarkImage(c *gin.Context) {
	var requestBody struct {
		ImageID string `json:"imageID"` // Expecting the Image ID to be sent in the POST request
	}

	// // Bind JSON request to requestBody struct
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := functions.ProcessSmallImageWithWatermark(requestBody.ImageID, StorageClient, FirestoreClient)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to Process"})
		return
	}
	latestStatus = fmt.Sprintf("small_watermarked_%s.jpg saved successfully", requestBody.ImageID)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}

func PostMediumWatermarkImage(c *gin.Context) {
	var requestBody struct {
		ImageID string `json:"imageID"` // Expecting the Image ID to be sent in the POST request
	}

	// // Bind JSON request to requestBody struct
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := functions.ProcessMediumImageWithWatermark(requestBody.ImageID, StorageClient, FirestoreClient)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to Process"})
		return
	}
	latestStatus = fmt.Sprintf("medium_watermarked_%s.jpg saved successfully", requestBody.ImageID)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}

func PostLargeWatermarkImage(c *gin.Context) {
	var requestBody struct {
		ImageID string `json:"imageID"` // Expecting the Image ID to be sent in the POST request
	}

	// // Bind JSON request to requestBody struct
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := functions.ProcessLargeImageWithWatermark(requestBody.ImageID, StorageClient, FirestoreClient)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to Process"})
		return
	}
	latestStatus = fmt.Sprintf("large_watermarked_%s.jpg saved successfully", requestBody.ImageID)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}
