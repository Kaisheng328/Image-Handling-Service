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
	latestStatus    string = "API is working fine !!!!"
)

func InitializeRoutes() {
	publicRoutes := Router.Group("v1/")
	publicRoutes.GET("health", HealthCheck)
	publicRoutes.POST("health", HandlePost)
	publicRoutes.POST("health/water", HandleWatermarkImage)

}

func InitializeClients() error {
	// Fetch credentials from Secret Manager for both local and cloud environments
	// credentialsJSON, err := FetchCredentialsFromSecretManager("projects/972298160089/secrets/kaisheng/versions/1")
	// if err != nil {
	// 	return fmt.Errorf("failed to fetch credentials: %v", err)
	// }

	// Use the retrieved credentials to initialize clients

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

func HealthCheck(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{
		"message": latestStatus,
	})
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

func HandlePost(c *gin.Context) {
	var requestBody struct {
		Base64Image string `json:"base64image"`
	}

	// ctx := context.Background()
	// sa := option.WithCredentialsFile("halogen-device-438608-v9-eeb6d5c67bff.json")
	// StorageClient, err := storage.NewClient(ctx, sa)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Credential File"})
	// 	fmt.Printf("error initializing app: %v\n", err)
	// 	return
	// }
	// defer StorageClient.Close()
	// // Initialize Firebase Storage
	// FirestoreClient, err := firestore.NewClient(ctx, "halogen-device-438608-v9", sa, option.WithEndpoint("asia-southeast1-firestore.googleapis.com:443"))
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Error initializing Firestore Client"})
	// 	fmt.Printf("Error initializing Firestore client: %v\n", err)
	// 	return
	// }
	// defer FirestoreClient.Close()

	// Parse the request body
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Println("Invalid Request Body")
		return
	}

	// Call the function to upload the image
	err := functions.UploadImageHandler(requestBody.Base64Image, StorageClient, FirestoreClient)
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

	// ctx := context.Background()
	// sa := option.WithCredentialsFile("halogen-device-438608-v9-eeb6d5c67bff.json")
	// StorageClient, err := storage.NewClient(ctx, sa)
	// if err != nil {
	// 	fmt.Printf("error initializing app: %v\n", err)
	// 	return
	// }
	// defer StorageClient.Close()
	// // Initialize Firebase Storage
	// FirestoreClient, err := firestore.NewClient(ctx, "halogen-device-438608-v9", sa, option.WithEndpoint("asia-southeast1-firestore.googleapis.com:443"))
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Error initializing Firestore Client"})
	// 	fmt.Printf("Error initializing Firestore client: %v\n", err)
	// 	return
	// }
	// defer FirestoreClient.Close()

	// // Bind JSON request to requestBody struct
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := functions.ProcessImageWithWatermark(requestBody.ImageID, StorageClient, FirestoreClient)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to Process"})
		return
	}
	latestStatus = fmt.Sprintf("watermarked_%s saved successfully", requestBody.ImageID)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}
