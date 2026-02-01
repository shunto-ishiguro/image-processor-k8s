package service

import (
	"image"
	"io"

	"github.com/disintegration/imaging"
)

// DecodeImage decodes an image from reader
func DecodeImage(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	return img, err
}

// Resize resizes the image to specified dimensions
// If width or height is 0, it preserves aspect ratio
func Resize(img image.Image, width, height int) image.Image {
	return imaging.Resize(img, width, height, imaging.Lanczos)
}

// Grayscale converts image to grayscale
func Grayscale(img image.Image) image.Image {
	return imaging.Grayscale(img)
}

// Rotate rotates image by specified angle (90, 180, 270)
func Rotate(img image.Image, angle int) image.Image {
	switch angle {
	case 90:
		return imaging.Rotate90(img)
	case 180:
		return imaging.Rotate180(img)
	case 270:
		return imaging.Rotate270(img)
	default:
		return img
	}
}
