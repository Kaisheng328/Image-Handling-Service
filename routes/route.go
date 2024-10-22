package routes

import (
	"Project/configs"
	"Project/functions"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	publicRoutes.GET("health/:id/:size", GetImagePath)
	publicRoutes.GET("health/:id/:size/water", GetWaterImagePath)
	publicRoutes.POST("uploadWatermark", PostWatermarkImage)
	publicRoutes.POST("health", PostImage)
	publicRoutes.POST("health/:size", PostImageResize)
	publicRoutes.POST("health/:size/water", PostImageWatermark)

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

func PostImage(c *gin.Context) {
	var requestBody struct {
		Base64Image string `json:"base64image"`
	}

	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Println("Invalid Request Body")
		return
	}
	location, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load timezone"})
		fmt.Println("Failed to load timezone")
		return
	}
	currentTime := time.Now().In(location)
	timestamp := currentTime.Format("20060102_150405")
	imageID := fmt.Sprintf("image_%s", timestamp)
	// Call the function to upload the image
	err = functions.UploadImageHandler(requestBody.Base64Image, StorageClient, FirestoreClient, timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	latestStatus = fmt.Sprintf("image_%v uploaded successfully", imageID)
	log.Printf("Image uploaded with ID: %s", imageID)
	c.JSON(http.StatusOK, gin.H{
		"status":  latestStatus,
		"imageID": imageID,
	})
}

func PostImageResize(c *gin.Context) {
	var requestBody struct {
		ImageID string `json:"imageID"` // Expecting the Image ID to be sent in the POST request
	}
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	sizename := c.Param("size")
	err := functions.ProcessResizeImage(requestBody.ImageID, sizename, StorageClient, FirestoreClient)
	if err != nil {
		log.Printf("Error in ProcessResizeImage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	latestStatus := fmt.Sprintf("%v resized to %v successfully", requestBody.ImageID, sizename)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}

func PostImageWatermark(c *gin.Context) {
	var requestBody struct {
		ImageID string `json:"imageID"` // Expecting the Image ID to be sent in the POST request
	}
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	sizename := c.Param("size")
	err := functions.ProcessImageWithWatermark(requestBody.ImageID, sizename, StorageClient, FirestoreClient)

	if err != nil {
		// c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to Process"})
		errResize := functions.ProcessResizeImage(requestBody.ImageID, sizename, StorageClient, FirestoreClient)
		if errResize != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to process resize"})
			return
		}
		errWatermark := functions.ProcessImageWithWatermark(requestBody.ImageID, sizename, StorageClient, FirestoreClient)
		if errWatermark != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to process watermark"})
			return
		}
		latestStatus := fmt.Sprintf("%v resized to %v and watermarked successfully after watermarking failed", requestBody.ImageID, sizename)
		c.JSON(http.StatusOK, gin.H{
			"status": latestStatus,
		})
		return
	}
	latestStatus = fmt.Sprintf("%s_watermarked_%s.jpg saved successfully", sizename, requestBody.ImageID)
	c.JSON(http.StatusOK, gin.H{
		"status": latestStatus,
	})
}

func GetImagePath(c *gin.Context) {
	ImageID := c.Param("id")
	sizename := c.Param("size")
	imageDetails, err := functions.GetImageDetailsFromFireStore(FirestoreClient, ImageID, sizename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Extract the image path from Firestore document
	imagePath, ok := imageDetails["Path"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Image path not found in Firestore document",
		})
		return
	}

	// Construct the Firebase Storage download URL (adjust the base URL as necessary)
	baseStorageURL := "https://firebasestorage.googleapis.com/v0/b/halogen-device-438608-v9.appspot.com/o/"
	fullURL := fmt.Sprintf("%s%s?alt=media", baseStorageURL, url.PathEscape(imagePath))

	// Fetch the image from Firebase Storage using the URL
	resp, err := http.Get(fullURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to download image from Firebase Storage",
		})
		return
	}
	defer resp.Body.Close()

	// Read the image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read image data",
		})
		return
	}

	// Set the content type to match the image type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream" // Fallback if content type is not available
	}

	// Send the image as the response
	c.Data(http.StatusOK, contentType, imageData)
}
func GetWaterImagePath(c *gin.Context) {
	ImageID := c.Param("id")
	sizename := c.Param("size")

	// Retrieve the image details from Firestore
	imageDetails, err := functions.GetWaterImageDetailFromFirestore(FirestoreClient, ImageID, sizename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Extract the image path from Firestore document
	imagePath, ok := imageDetails["Path"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Image path not found in Firestore document",
		})
		return
	}

	// Construct the Firebase Storage download URL (adjust the base URL as necessary)
	baseStorageURL := "https://firebasestorage.googleapis.com/v0/b/halogen-device-438608-v9.appspot.com/o/"
	fullURL := fmt.Sprintf("%s%s?alt=media", baseStorageURL, url.PathEscape(imagePath))

	// Fetch the image from Firebase Storage using the URL
	resp, err := http.Get(fullURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to download image from Firebase Storage",
		})
		return
	}
	defer resp.Body.Close()

	// Read the image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read image data",
		})
		return
	}

	// Set the content type to match the image type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream" // Fallback if content type is not available
	}

	// Send the image as the response
	c.Data(http.StatusOK, contentType, imageData)
}
func PostWatermarkImage(c *gin.Context) {
	var requestBody struct {
		Base64Image string `json:"base64image"`
		ImageName   string `json:"imagename"` // Expect the name to be provided in the request body
	}

	// Bind the JSON request to the requestBody struct
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Println("Invalid Request Body")
		return
	}

	// Validate that the image name is provided
	if requestBody.ImageName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image name is required"})
		fmt.Println("Image name not provided")
		return
	}

	// Call the function to upload the watermark image
	err := functions.UploadWatermarkImageHandler(requestBody.Base64Image, requestBody.ImageName, StorageClient, FirestoreClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	latestStatus := fmt.Sprintf("Watermark image %v uploaded successfully", requestBody.ImageName)
	log.Printf("Watermark image uploaded with name: %s", requestBody.ImageName)

	// Send success response with the provided image name
	c.JSON(http.StatusOK, gin.H{
		"status":    latestStatus,
		"imageName": requestBody.ImageName,
	})
}
