package functions

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/disintegration/imaging"
	"github.com/nfnt/resize"
)

type ImageDocument struct {
	ID          string `firestore:"id"`
	Description string `firestore:"description"`
	smallPath   string `firestore:"smallPath"`
	mediumPath  string `firestore:"mediumPath"`
	largePath   string `firestore:"largePath"`
	Path        string `firestore:"Path"`
}

func SaveImageDetailsToFirestore(client *firestore.Client, id, description, smallPath, mediumPath, largePath string) error {
	ctx := context.Background()

	// Reference the Firestore collection
	docRef := client.Collection("posts").Doc(id)

	// Define the document structure
	imageDoc := map[string]interface{}{
		"ID":          id,
		"Description": description,
		"smallPath":   smallPath,
		"mediumPath":  mediumPath,
		"largePath":   largePath,
	}

	// Write to Firestore
	_, err := docRef.Set(ctx, imageDoc)
	if err != nil {
		return fmt.Errorf("failed to save image details to Firestore: %v", err)
	}

	log.Printf("Image details saved to Firestore: ID = %s\n", id)
	return nil
}

func SaveWatermarkedImageDetailsToFirestore(client *firestore.Client, parentID, watermarkID, description, path string) error {
	ctx := context.Background()

	// Reference the Firestore collection
	docRef := client.Collection("posts").Doc(parentID).Collection("watermarks").Doc(watermarkID)

	// Define the document structure
	WatermarkDoc := map[string]interface{}{
		"ID":          watermarkID,
		"Description": description,
		"Path":        path,
	}

	// Write to Firestore
	_, err := docRef.Set(ctx, WatermarkDoc)
	if err != nil {
		return fmt.Errorf("failed to save image details to Firestore: %v", err)
	}

	log.Printf("Watermarked image details saved to Firestore: parentID = %s, watermarkID = %s\n", parentID, watermarkID)
	return nil
}
func CalculateWatermarkPositions(imgWidth, imgHeight, wmWidth, wmHeight, numWatermarks int) []image.Point {
	var positions []image.Point

	// Define grid size (e.g., 2x2, 3x3, depending on the number of watermarks)
	gridCols := 2
	gridRows := (numWatermarks + 1) / gridCols

	// Calculate padding between watermarks and edges
	colPadding := (imgWidth - gridCols*wmWidth) / (gridCols + 1)
	rowPadding := (imgHeight - gridRows*wmHeight) / (gridRows + 1)

	// Generate watermark positions in a grid pattern
	for row := 0; row < gridRows; row++ {
		for col := 0; col < gridCols; col++ {
			if len(positions) < numWatermarks {
				x := colPadding + col*(wmWidth+colPadding)
				y := rowPadding + row*(wmHeight+rowPadding)
				positions = append(positions, image.Pt(x, y))
			}
		}
	}

	return positions
}
func ResizeSmallImage(img image.Image) image.Image {
	small := resize.Resize(100, 0, img, resize.Lanczos3)
	return small
}
func ResizeMediumImage(img image.Image) image.Image {
	medium := resize.Resize(500, 0, img, resize.Lanczos3)
	return medium
}
func ResizeLargeImage(img image.Image) image.Image {
	large := resize.Resize(1500, 0, img, resize.Lanczos3)
	return large
}
func AddWatermark(img image.Image, watermark image.Image) image.Image {
	imgWidth := img.Bounds().Dx()
	imgHeight := img.Bounds().Dy()

	// Define the number of watermarks based on image size
	var numWatermarks int
	if imgWidth < 500 { // Small image
		numWatermarks = 1
	} else if imgWidth < 1000 { // Medium image
		numWatermarks = 2
	} else { // Large image
		numWatermarks = 5
	}

	// Calculate proportional watermark size and resize it
	watermarkWidth := int(float64(imgWidth) * 0.2) // 20% of the image width
	watermark = imaging.Resize(watermark, watermarkWidth, 0, imaging.Lanczos)
	alpha := 0.7
	watermarkedImage := AddTransparency(watermark, alpha)
	// Create a new image to hold the final result
	finalImg := image.NewRGBA(img.Bounds())
	draw.Draw(finalImg, finalImg.Bounds(), img, image.Point{}, draw.Src)

	// Generate positions for the watermarks
	positions := CalculateWatermarkPositions(imgWidth, imgHeight, watermark.Bounds().Dx(), watermark.Bounds().Dy(), numWatermarks)

	// Draw the watermark at each position
	for _, pos := range positions {
		draw.Draw(finalImg, watermark.Bounds().Add(pos), watermarkedImage, image.Point{}, draw.Over)
	}

	return finalImg
}
func AddTransparency(img image.Image, alpha float64) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			_, _, _, a := originalColor.RGBA()

			// Only apply transparency if the pixel is not fully transparent
			if a > 0 {
				r, g, b, _ := originalColor.RGBA()
				transparentColor := color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: uint8(alpha * 255),
				}
				rgba.Set(x, y, transparentColor)
			}
		}
	}
	return rgba
}
func UploadImageToFirebase(client *storage.Client, filename string, img image.Image) (string, error) {
	ctx := context.Background()

	// Create a bucket reference
	bucketName := "halogen-device-438608-v9.appspot.com" // Replace with your bucket name
	bucket := client.Bucket(bucketName)

	// Create a new file in the bucket
	obj := bucket.Object(filename)
	writer := obj.NewWriter(ctx)
	defer writer.Close()

	// Encode and write the image to Firebase Storage as JPEG
	err := jpeg.Encode(writer, img, nil)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func GetSmallImageDetailFromFirestore(client *firestore.Client, parentID string) (map[string]interface{}, error) {
	ctx := context.Background()

	// Reference the Firestore document for the small image
	docRef := client.Collection("posts").Doc(parentID).Collection("resized_images").Doc("small")

	// Get the document from Firestore
	doc, err := docRef.Get(ctx)
	if err != nil {

		return nil, fmt.Errorf("failed to get small image details from Firestore: %v", err)
	}

	// Extract the document data
	imageDetails := doc.Data()

	log.Printf("Small image details retrieved from Firestore: parentID = %s\n", parentID)
	return imageDetails, nil
}
func GetMediumImageDetailFromFirestore(client *firestore.Client, parentID string) (map[string]interface{}, error) {
	ctx := context.Background()

	// Reference the Firestore document for the small image
	docRef := client.Collection("posts").Doc(parentID).Collection("resized_images").Doc("medium")

	// Get the document from Firestore
	doc, err := docRef.Get(ctx)
	if err != nil {

		return nil, fmt.Errorf("failed to get medium image details from Firestore: %v", err)
	}

	// Extract the document data
	imageDetails := doc.Data()

	log.Printf("Medium image details retrieved from Firestore: parentID = %s\n", parentID)
	return imageDetails, nil
}
func GetLargeImageDetailFromFirestore(client *firestore.Client, parentID string) (map[string]interface{}, error) {
	ctx := context.Background()

	// Reference the Firestore document for the small image
	docRef := client.Collection("posts").Doc(parentID).Collection("resized_images").Doc("large")

	// Get the document from Firestore
	doc, err := docRef.Get(ctx)
	if err != nil {

		return nil, fmt.Errorf("failed to get large image details from Firestore: %v", err)
	}

	// Extract the document data
	imageDetails := doc.Data()

	log.Printf("large image details retrieved from Firestore: parentID = %s\n", parentID)
	return imageDetails, nil
}

func SaveUploadedImageDetailsToFirestore(client *firestore.Client, id, description, Filepath string) error {
	ctx := context.Background()

	// Reference the Firestore collection
	docRef := client.Collection("posts").Doc(id)

	// Define the document structure
	imageDoc := map[string]interface{}{
		"ID":          id,
		"Description": description,
		"Filepath":    Filepath,
	}

	// Write to Firestore
	_, err := docRef.Set(ctx, imageDoc)
	if err != nil {
		return fmt.Errorf("failed to save image details to Firestore: %v", err)
	}

	log.Printf("Image details saved to Firestore: ID = %s\n", id)
	return nil
}
func UploadImageHandler(base64ImageData string, StorageClient *storage.Client, firestoreClient *firestore.Client, timestamp string) error {
	if strings.HasPrefix(base64ImageData, "data:image/") {
		commaIndex := strings.Index(base64ImageData, ",")
		if commaIndex != -1 {
			base64ImageData = base64ImageData[commaIndex+1:]
		}
	}

	// Decode the Base64 string into image bytes
	imageData, err := base64.StdEncoding.DecodeString(base64ImageData)
	if err != nil {
		return fmt.Errorf("unable to decode Base64 string: %v", err)
	}

	// Create a reader from the decoded bytes
	imageReader := strings.NewReader(string(imageData))

	// Decode the image to check if it's a valid image
	img, format, err := image.Decode(imageReader)
	if err != nil {
		return fmt.Errorf("invalid image format: %v", err)
	}

	log.Printf("Image decoded successfully: format = %s\n", format)
	Filename := fmt.Sprintf("image_%s.jpg", timestamp)
	Filepath, err := UploadImageToFirebase(StorageClient, Filename, img)
	if err != nil {
		return fmt.Errorf("error uploading image: %v", err)
	}
	ID := fmt.Sprintf("image_%s", timestamp)
	description := "Image uploaded successfully!!!"
	err = SaveUploadedImageDetailsToFirestore(firestoreClient, ID, description, Filepath)
	if err != nil {
		return fmt.Errorf("error saving image details to Firestore: %v", err)
	}
	return nil

}

func SaveResizedImageDetailsToFirestore(client *firestore.Client, parentID, sizeID, description, path string) error {
	ctx := context.Background()

	// Reference the Firestore collection
	docRef := client.Collection("posts").Doc(parentID).Collection("resized_images").Doc(sizeID)

	// Define the document structure
	resizedImageDoc := map[string]interface{}{
		"ID":          sizeID,
		"Description": description,
		"Path":        path,
	}

	// Write to Firestore
	_, err := docRef.Set(ctx, resizedImageDoc)
	if err != nil {
		return fmt.Errorf("failed to save resized image details to Firestore: %v", err)
	}

	log.Printf("Resized image details saved to Firestore: parentID = %s, sizeID = %s\n", parentID, sizeID)
	return nil
}
func ProcessResizeSmallImage(imageID string, StorageClient *storage.Client, firestoreClient *firestore.Client) error {
	ctx := context.Background()
	objectPath := fmt.Sprintf("%s.jpg", imageID)
	bucketName := "halogen-device-438608-v9.appspot.com"
	reader, err := StorageClient.Bucket(bucketName).Object(objectPath).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get image from storage: %v", err)
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	// Step 2: Resize the image to a small size
	smallImage := ResizeSmallImage(img)

	// Step 3: Upload the resized image back to Firebase Storage
	smallPath := fmt.Sprintf("resized/small_%s.jpg", imageID)
	writer := StorageClient.Bucket(bucketName).Object(smallPath).NewWriter(ctx)
	defer writer.Close()

	writer.ContentType = "image/jpeg"
	if err := jpeg.Encode(writer, smallImage, &jpeg.Options{Quality: 90}); err != nil {
		return fmt.Errorf("failed to encode and upload resized image: %v", err)
	}

	// Step 4: Save the resized image details to Firestore with the original image ID as the parentID
	err = SaveResizedImageDetailsToFirestore(firestoreClient, imageID, "small", "Small size image", smallPath)
	if err != nil {
		return fmt.Errorf("failed to save resized image details to Firestore: %v", err)
	}

	fmt.Println("Resized image saved successfully:", smallPath)
	return nil
}
func ProcessResizeMediumImage(imageID string, StorageClient *storage.Client, firestoreClient *firestore.Client) error {
	ctx := context.Background()
	objectPath := fmt.Sprintf("%s.jpg", imageID)
	bucketName := "halogen-device-438608-v9.appspot.com"
	reader, err := StorageClient.Bucket(bucketName).Object(objectPath).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get image from storage: %v", err)
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	// Step 2: Resize the image to a small size
	mediumImage := ResizeMediumImage(img)

	// Step 3: Upload the resized image back to Firebase Storage
	mediumPath := fmt.Sprintf("resized/medium_%s.jpg", imageID)
	writer := StorageClient.Bucket(bucketName).Object(mediumPath).NewWriter(ctx)
	defer writer.Close()

	writer.ContentType = "image/jpeg"
	if err := jpeg.Encode(writer, mediumImage, &jpeg.Options{Quality: 90}); err != nil {
		return fmt.Errorf("failed to encode and upload resized image: %v", err)
	}

	// Step 4: Save the resized image details to Firestore with the original image ID as the parentID
	err = SaveResizedImageDetailsToFirestore(firestoreClient, imageID, "medium", "medium size image", mediumPath)
	if err != nil {
		return fmt.Errorf("failed to save resized image details to Firestore: %v", err)
	}

	fmt.Println("Resized image saved successfully:", mediumPath)
	return nil
}

func ProcessResizeLargeImage(imageID string, StorageClient *storage.Client, firestoreClient *firestore.Client) error {
	ctx := context.Background()
	objectPath := fmt.Sprintf("%s.jpg", imageID)
	bucketName := "halogen-device-438608-v9.appspot.com"
	reader, err := StorageClient.Bucket(bucketName).Object(objectPath).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get image from storage: %v", err)
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	// Step 2: Resize the image to a small size
	largeImage := ResizeLargeImage(img)

	// Step 3: Upload the resized image back to Firebase Storage
	largePath := fmt.Sprintf("resized/large_%s.jpg", imageID)
	writer := StorageClient.Bucket(bucketName).Object(largePath).NewWriter(ctx)
	defer writer.Close()

	writer.ContentType = "image/jpeg"
	if err := jpeg.Encode(writer, largeImage, &jpeg.Options{Quality: 90}); err != nil {
		return fmt.Errorf("failed to encode and upload resized image: %v", err)
	}

	// Step 4: Save the resized image details to Firestore with the original image ID as the parentID
	err = SaveResizedImageDetailsToFirestore(firestoreClient, imageID, "large", "large size image", largePath)
	if err != nil {
		return fmt.Errorf("failed to save resized image details to Firestore: %v", err)
	}

	fmt.Println("Resized image saved successfully:", largePath)
	return nil
}

func ProcessSmallImageWithWatermark(imageID string, storageClient *storage.Client, firestoreClient *firestore.Client) error {
	ctx := context.Background()
	bucketName := "halogen-device-438608-v9.appspot.com"
	// Step 1: Find the small resized image path from Firestore
	docRef := firestoreClient.Collection("posts").Doc(imageID).Collection("resized_images").Doc("small")
	doc, err := docRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get small resized image from Firestore: %v", err)
	}

	// Get the image path
	smallImagePath, ok := doc.Data()["Path"].(string)
	if !ok {
		return fmt.Errorf("failed to find the 'Path' field in the Firestore document")
	}

	// Step 2: Download the small resized image from Firebase Storage
	reader, err := storageClient.Bucket(bucketName).Object(smallImagePath).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to download small image from storage: %v", err)
	}
	defer reader.Close()

	smallimg, _, err := image.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode small image: %v", err)
	}

	// Step 3: Download the watermark image from Firebase Storage
	watermark, err := imaging.Open("Icares_Logo.png")
	if err != nil {
		return fmt.Errorf("watermark loading failed: %v", err)
	}

	// Step 4: Apply the watermark on the small image
	imgWithWatermark := AddWatermark(smallimg, watermark)

	// Step 5: Save the watermarked image back to Firebase Storage
	watermarkedPath := fmt.Sprintf("watermarked/small_watermarked_%s.jpg", imageID)
	writer := storageClient.Bucket(bucketName).Object(watermarkedPath).NewWriter(ctx)
	defer writer.Close()

	writer.ContentType = "image/jpeg"
	if err := jpeg.Encode(writer, imgWithWatermark, &jpeg.Options{Quality: 90}); err != nil {
		return fmt.Errorf("failed to encode and upload watermarked image: %v", err)
	}

	// Step 6: Save the watermarked image path to Firestore
	err = SaveWatermarkedImageDetailsToFirestore(firestoreClient, imageID, "watermarked_small", "Watermarked small image", watermarkedPath)
	if err != nil {
		return fmt.Errorf("failed to save watermarked image details to Firestore: %v", err)
	}

	fmt.Printf("Watermarked small image saved successfully: %s\n", watermarkedPath)
	return nil
}

func ProcessMediumImageWithWatermark(imageID string, storageClient *storage.Client, firestoreClient *firestore.Client) error {
	ctx := context.Background()
	bucketName := "halogen-device-438608-v9.appspot.com"
	docRef := firestoreClient.Collection("posts").Doc(imageID).Collection("resized_images").Doc("medium")
	doc, err := docRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get medium resized image from Firestore: %v", err)
	}
	mediumImagePath, ok := doc.Data()["Path"].(string)
	if !ok {
		return fmt.Errorf("failed to find the 'Path' field in the Firestore document")
	}

	reader, err := storageClient.Bucket(bucketName).Object(mediumImagePath).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to download medium image from storage: %v", err)
	}
	defer reader.Close()

	mediumimg, _, err := image.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode small image: %v", err)
	}

	// Step 3: Download the watermark image from Firebase Storage
	watermark, err := imaging.Open("Icares_Logo.png")
	if err != nil {
		return fmt.Errorf("watermark loading failed: %v", err)
	}

	imgWithWatermark := AddWatermark(mediumimg, watermark)
	watermarkedPath := fmt.Sprintf("watermarked/medium_watermarked_%s.jpg", imageID)
	writer := storageClient.Bucket(bucketName).Object(watermarkedPath).NewWriter(ctx)
	defer writer.Close()

	writer.ContentType = "image/jpeg"
	if err := jpeg.Encode(writer, imgWithWatermark, &jpeg.Options{Quality: 90}); err != nil {
		return fmt.Errorf("failed to encode and upload watermarked image: %v", err)
	}
	err = SaveWatermarkedImageDetailsToFirestore(firestoreClient, imageID, "watermarked_medium", "Watermarked medium image", watermarkedPath)
	if err != nil {
		return fmt.Errorf("failed to save watermarked image details to Firestore: %v", err)
	}

	fmt.Printf("Watermarked medium image saved successfully: %s\n", watermarkedPath)
	return nil
}

func ProcessLargeImageWithWatermark(imageID string, storageClient *storage.Client, firestoreClient *firestore.Client) error {
	ctx := context.Background()
	bucketName := "halogen-device-438608-v9.appspot.com"
	// Step 1: Find the small resized image path from Firestore
	docRef := firestoreClient.Collection("posts").Doc(imageID).Collection("resized_images").Doc("large")
	doc, err := docRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get medium resized image from Firestore: %v", err)
	}

	// Get the image path
	largeImagePath, ok := doc.Data()["Path"].(string)
	if !ok {
		return fmt.Errorf("failed to find the 'Path' field in the Firestore document")
	}

	// Step 2: Download the small resized image from Firebase Storage
	reader, err := storageClient.Bucket(bucketName).Object(largeImagePath).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to download medium image from storage: %v", err)
	}
	defer reader.Close()

	largeimg, _, err := image.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode small image: %v", err)
	}

	// Step 3: Download the watermark image from Firebase Storage
	watermark, err := imaging.Open("Icares_Logo.png")
	if err != nil {
		return fmt.Errorf("watermark loading failed: %v", err)
	}

	// Step 4: Apply the watermark on the small image
	imgWithWatermark := AddWatermark(largeimg, watermark)

	// Step 5: Save the watermarked image back to Firebase Storage
	watermarkedPath := fmt.Sprintf("watermarked/large_watermarked_%s.jpg", imageID)
	writer := storageClient.Bucket(bucketName).Object(watermarkedPath).NewWriter(ctx)
	defer writer.Close()

	writer.ContentType = "image/jpeg"
	if err := jpeg.Encode(writer, imgWithWatermark, &jpeg.Options{Quality: 90}); err != nil {
		return fmt.Errorf("failed to encode and upload watermarked image: %v", err)
	}

	// Step 6: Save the watermarked image path to Firestore
	err = SaveWatermarkedImageDetailsToFirestore(firestoreClient, imageID, "watermarked_large", "Watermarked large image", watermarkedPath)
	if err != nil {
		return fmt.Errorf("failed to save watermarked image details to Firestore: %v", err)
	}

	fmt.Printf("Watermarked large image saved successfully: %s\n", watermarkedPath)
	return nil
}

func GetSmallWaterImageDetailFromFirestore(client *firestore.Client, parentID string) (map[string]interface{}, error) {
	ctx := context.Background()

	// Reference the Firestore document for the small image
	docRef := client.Collection("posts").Doc(parentID).Collection("watermarks").Doc("watermarked_small")

	// Get the document from Firestore
	doc, err := docRef.Get(ctx)
	if err != nil {

		return nil, fmt.Errorf("failed to get small watermark image details from Firestore: %v", err)
	}

	// Extract the document data
	imageDetails := doc.Data()

	log.Printf("Small Watermark image details retrieved from Firestore: parentID = %s\n", parentID)
	return imageDetails, nil
}
func GetMediumWaterImageDetailFromFirestore(client *firestore.Client, parentID string) (map[string]interface{}, error) {
	ctx := context.Background()

	// Reference the Firestore document for the small image
	docRef := client.Collection("posts").Doc(parentID).Collection("watermarks").Doc("watermarked_medium")

	// Get the document from Firestore
	doc, err := docRef.Get(ctx)
	if err != nil {

		return nil, fmt.Errorf("failed to get medium watermark image details from Firestore: %v", err)
	}

	// Extract the document data
	imageDetails := doc.Data()

	log.Printf("medium Watermark image details retrieved from Firestore: parentID = %s\n", parentID)
	return imageDetails, nil
}
func GetLargeWaterImageDetailFromFirestore(client *firestore.Client, parentID string) (map[string]interface{}, error) {
	ctx := context.Background()

	// Reference the Firestore document for the small image
	docRef := client.Collection("posts").Doc(parentID).Collection("watermarks").Doc("watermarked_large")

	// Get the document from Firestore
	doc, err := docRef.Get(ctx)
	if err != nil {

		return nil, fmt.Errorf("failed to get large watermark image details from Firestore: %v", err)
	}

	// Extract the document data
	imageDetails := doc.Data()

	log.Printf("Large Watermark image details retrieved from Firestore: parentID = %s\n", parentID)
	return imageDetails, nil
}
