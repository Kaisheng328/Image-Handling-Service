document.getElementById('uploadButton').addEventListener('click', async function () {
    const fileInput = document.getElementById('imageUpload').files[0];

    if (!fileInput) {
        alert("Please select an image");
        return;
    }

    const reader = new FileReader();
    reader.onloadend = async function () {
        const base64Image = reader.result;

        const requestBody = {
            base64image: base64Image,
        };

        const response = await fetch('/v1/health', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
            alert('Error uploading the image');
            return;
        }

        const result = await response.json();
        console.log("Upload result:", result);  // Debugging: Check the response from the server

        if (result.imageID) {
            // Store the new imageID in localStorage after uploading the new image
            localStorage.setItem('imageID', result.imageID);
            alert(result.status);
        } else {
            console.error("No imageID returned in the response.");
            alert("Error: No image ID received after upload.");
        }

        // Reset the file input to allow further uploads
        document.getElementById('imageUpload').value = "";
    };

    reader.readAsDataURL(fileInput);
});

document.getElementById('resizeButton').addEventListener('click', async function () {
    const size = document.getElementById('sizeSelect').value;
    const imageID = localStorage.getItem('imageID'); // Retrieve the image ID from local storage

    if (!imageID) {
        alert("No image ID found. Please upload an image first.");
        return;
    }

    const requestBody = {
        imageID: imageID,
    };

    const response = await fetch(`/v1/health/${size}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
    });

    const result = await response.json();
    console.log("Resize result:", result);  // Debugging: Check the response from the server

    if (response.ok) {
        alert(result.status); // Show success message

        
    } else {
        alert(`Error resizing the image: ${result.error || 'Unknown error'}`);
        console.error(result); // Log the entire response for debugging
    }
});

document.getElementById('uploadWatermarkButton').addEventListener('click', async function () {
    const watermarkFile = document.getElementById('watermarkUpload').files[0];
    const watermarkImageName = document.getElementById('watermarkImageName').value;

    if (!watermarkFile) {
        alert("Please select a watermark image");
        return;
    }

    if (!watermarkImageName) {
        alert("Please enter a watermark image name");
        return;
    }

    const reader = new FileReader();
    reader.onloadend = async function () {
        const base64Watermark = reader.result.split(",")[1]; // Get base64 part of the image

        const requestBody = {
            base64image: base64Watermark, // Base64-encoded watermark image
            imagename: watermarkImageName // Watermark image name entered by the user
        };

        const response = await fetch('/v1/uploadWatermark', { // Replace with your actual route
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestBody),
        });

        if (!response.ok) {
            alert('Error uploading the watermark image');
            return;
        }

        const result = await response.json();
        console.log("Watermark Upload result:", result);  // Debugging: Check the response from the server

        alert(result.status); // Show success message from backend

        // Optionally, clear input fields
        document.getElementById('watermarkUpload').value = "";
        document.getElementById('watermarkImageName').value = "";
    };

    reader.readAsDataURL(watermarkFile); // Convert the image file to base64
});


document.getElementById('applyWatermarkButton').addEventListener('click', async function () {
    const size = document.getElementById('sizeSelect').value; // Get the selected size
    const imageID = localStorage.getItem('imageID'); // Retrieve the image ID from local storage

    if (!imageID) {
        alert("No image ID found. Please upload and resize an image first.");
        return;
    }

    const requestBody = {
        imageID: imageID,
    };

    // Make the request to apply the watermark to the resized image
    const response = await fetch(`/v1/health/${size}/water`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
    });

    const result = await response.json();
    console.log("Apply watermark result:", result);  // Debugging: Check the response from the server

    if (response.ok) {
        alert(result.status); // Show success message
    } else {
        alert(`Error applying watermark: ${result.error || 'Unknown error'}`);
        console.error(result); // Log the entire response for debugging
    }
});
document.getElementById('previewResizeButton').addEventListener('click', async function () {
    const imageID = localStorage.getItem('imageID'); // Assuming the image ID is saved here
    const size = document.getElementById('sizeSelect').value; // Get the size from the dropdown

    if (!imageID) {
        alert("No image ID found. Please upload an image first.");
        return;
    }

    try {
        // Fetch the image from the backend
        const response = await fetch(`/v1/health/${imageID}/${size}`, {
            method: 'GET',
        });

        if (!response.ok) {
            alert('Error fetching the image');
            return;
        }

        // Read the image blob from the response
        const imageBlob = await response.blob();

        // Create a URL for the blob and set it as the src of the image element
        const imageUrl = URL.createObjectURL(imageBlob);
        const imagePreview = document.getElementById('imagePreview');
        imagePreview.src = imageUrl;

        // Show the image and the close button
        imagePreview.style.display = 'block';
        document.getElementById('closePreviewResizeButton').style.display = 'block';

        console.log('Image previewed successfully');
    } catch (error) {
        console.error('Error fetching or displaying the image:', error);
        alert('Failed to preview image');
    }
});

// Close Preview Button Logic
document.getElementById('closePreviewResizeButton').addEventListener('click', function () {
    const imagePreview = document.getElementById('imagePreview');
    imagePreview.style.display = 'none'; // Hide the image
    document.getElementById('closePreviewResizeButton').style.display = 'none'; // Hide the close button
});

document.getElementById('previewWatermarkButton').addEventListener('click', async function () {
    const imageID = localStorage.getItem('imageID'); // Assuming the image ID is saved here
    const size = document.getElementById('sizeSelect').value; // Get the size from the dropdown

    if (!imageID) {
        alert("No image ID found. Please upload an image first.");
        return;
    }

    try {
        // Fetch the image from the backend
        const response = await fetch(`/v1/health/${imageID}/${size}/water`, {
            method: 'GET',
        });

        if (!response.ok) {
            alert('Error fetching the image');
            return;
        }

        // Read the image blob from the response
        const imageBlob = await response.blob();

        // Create a URL for the blob and set it as the src of the image element
        const imageUrl = URL.createObjectURL(imageBlob);
        const imagePreview = document.getElementById('WaterimagePreview');
        imagePreview.src = imageUrl;

        // Show the image and the close button
        imagePreview.style.display = 'block';
        document.getElementById('closePreviewWaterButton').style.display = 'block';

        console.log('Image previewed successfully');
    } catch (error) {
        console.error('Error fetching or displaying the image:', error);
        alert('Failed to preview image');
    }
});

document.getElementById('closePreviewWaterButton').addEventListener('click', function () {
    const imagePreview = document.getElementById('WaterimagePreview');
    imagePreview.style.display = 'none'; // Hide the image
    document.getElementById('closePreviewWaterButton').style.display = 'none'; // Hide the close button
});