package qrcode

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/nfnt/resize"
	"github.com/nxshock/colorcrop"
	"github.com/rs/zerolog/log"
)

func pngToGrayScale(imageBytes []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		log.Error().Err(err).Msg("failed to decode PNG image")
	}

	// 4. Create a new grayscale image with the same bounds as the original
	bounds := img.Bounds()
	grayImage := image.NewGray(bounds)

	// 5. Draw the original image onto the grayscale image.
	// The draw.Draw function automatically handles the conversion to grayscale.
	draw.Draw(grayImage, bounds, img, bounds.Min, draw.Src)

	//// encode back to bytes
	var buf bytes.Buffer
	err = png.Encode(&buf, grayImage)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode resized PNG image")
	}
	return buf.Bytes(), err
}

func pngResize(imageBytes []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		log.Error().Err(err).Msg("failed to decode PNG image")
	}
	resizedImg := resize.Resize(90, 90, img, resize.Lanczos3)

	//// encode back to bytes
	var buf bytes.Buffer
	err = png.Encode(&buf, resizedImg)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode resized PNG image")
	}
	return buf.Bytes(), err
}

func pngCropBorder(imageBytes []byte) ([]byte, error) {
	// generated via gemini
	img, err := png.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		log.Error().Err(err).Msg("failed to decode PNG image")
	}

	croppedImage := colorcrop.Crop(
		img,
		color.RGBA{255, 255, 255, 255}, // crop white border
		0.5,                            // with 50% threshold
	)

	// add white border margin
	margin := 50
	originalBounds := croppedImage.Bounds()
	originalWidth := originalBounds.Dx()
	originalHeight := originalBounds.Dy()

	// Calculate new dimensions with margin
	newWidth := originalWidth + 2*margin
	newHeight := originalHeight + 2*margin

	// Create a new RGBA image with the calculated new dimensions
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Fill the new image with white color (the margin)
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			newImg.Set(x, y, color.White)
		}
	}

	// Draw the original image onto the center of the new image
	drawPoint := image.Point{margin, margin}
	for y := 0; y < originalHeight; y++ {
		for x := 0; x < originalWidth; x++ {
			newImg.Set(drawPoint.X+x, drawPoint.Y+y, croppedImage.At(originalBounds.Min.X+x, originalBounds.Min.Y+y))
		}
	}

	//// encode back to bytes
	var buf bytes.Buffer
	err = png.Encode(&buf, newImg)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode resized PNG image")
	}
	return buf.Bytes(), err
}
