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
	"time"

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
func ResizeImage(img image.Image) (image.Image, image.Image, image.Image) {
	small := resize.Resize(100, 0, img, resize.Lanczos3)
	medium := resize.Resize(500, 0, img, resize.Lanczos3)
	large := resize.Resize(1500, 0, img, resize.Lanczos3)
	return small, medium, large
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
func GetImageDetailsFromFirestore(client *firestore.Client, id string) (string, string, string, error) {
	ctx := context.Background()

	// Reference the Firestore document by ID
	docRef := client.Collection("posts").Doc(id)
	doc, err := docRef.Get(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get document: %v", err)
	}

	// Extract the image paths from the document
	smallPath := doc.Data()["smallPath"].(string)
	mediumPath := doc.Data()["mediumPath"].(string)
	largePath := doc.Data()["largePath"].(string)

	return smallPath, mediumPath, largePath, nil
}

func ProcessImageWithWatermark(imageID string, StorageClient *storage.Client, firestoreClient *firestore.Client) error {

	_, mediumPath, _, err := GetImageDetailsFromFirestore(firestoreClient, imageID)
	if err != nil {
		return fmt.Errorf("error retrieving image details from Firestore: %v", err)
	}

	img, err := DownloadImageFromFirebase(StorageClient, mediumPath)
	if err != nil {
		return fmt.Errorf("error downloading image: %v", err)
	}

	watermark, err := imaging.Open("Icares_Logo.png")
	if err != nil {
		return fmt.Errorf("Watermark loading failed: %v", err)
	}
	imgWithWatermark := AddWatermark(img, watermark)

	watermarkedFilename := fmt.Sprintf("watermarked_image_%s.jpg", imageID)

	watermarkedPath, err := UploadImageToFirebase(StorageClient, watermarkedFilename, imgWithWatermark)
	if err != nil {
		return fmt.Errorf("error uploading watermarked image: %v", err)
	}

	// Step 5: Save the new image details to Firestore in a new collection
	watermarkID := fmt.Sprintf("watermarked_%s", imageID)
	description := "Watermarked image"
	err = SaveWatermarkedImageDetailsToFirestore(firestoreClient, imageID, watermarkID, description, watermarkedPath)
	if err != nil {
		return fmt.Errorf("error saving watermarked image details to Firestore: %v", err)
	}

	log.Println("Watermarked image processed and saved successfully as a child document")
	return nil
}
func DownloadImageFromFirebase(StorageClient *storage.Client, filePath string) (image.Image, error) {
	ctx := context.Background()

	// Reference the bucket and the object (image file)
	bucket := StorageClient.Bucket("halogen-device-438608-v9.appspot.com")
	obj := bucket.Object(filePath)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %v", err)
	}
	defer reader.Close()

	// Decode the image
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	return img, nil
}

func UploadImageHandler(base64ImageData string, StorageClient *storage.Client, firestoreClient *firestore.Client) error {
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

	currentTime := time.Now()
	timestamp := currentTime.Format("20060102_150405")

	smallFilename := fmt.Sprintf("image_%s_small.jpg", timestamp)
	mediumFilename := fmt.Sprintf("image_%s_medium.jpg", timestamp)
	largeFilename := fmt.Sprintf("image_%s_large.jpg", timestamp)
	small, medium, large := ResizeImage(img)
	smallPath, err := UploadImageToFirebase(StorageClient, smallFilename, small)
	if err != nil {
		return fmt.Errorf("error uploading small image: %v", err)
	}

	mediumPath, err := UploadImageToFirebase(StorageClient, mediumFilename, medium)
	if err != nil {
		return fmt.Errorf("error uploading medium image: %v", err)
	}

	largePath, err := UploadImageToFirebase(StorageClient, largeFilename, large)
	if err != nil {
		return fmt.Errorf("error uploading large image: %v", err)
	}
	ID := fmt.Sprintf("image_%s", timestamp)
	description := "Image uploaded successfully!"
	err = SaveImageDetailsToFirestore(firestoreClient, ID, description, smallPath, mediumPath, largePath)
	if err != nil {
		return fmt.Errorf("error saving image details to Firestore: %v", err)
	}
	return nil
}
