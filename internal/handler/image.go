package handler

import (
	"bytes"
	"encoding/json"
	"image/jpeg"
	"image/png"
	"net/http"
	"strconv"

	"github.com/shunto/image-processing-k8s/internal/service"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func writeImage(w http.ResponseWriter, buf *bytes.Buffer, format string) {
	switch format {
	case "png":
		w.Header().Set("Content-Type", "image/png")
	default:
		w.Header().Set("Content-Type", "image/jpeg")
	}
	w.Write(buf.Bytes())
}

// Resize handles image resize requests
// Query params: width, height, format (jpeg/png)
func Resize(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, "image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	width, _ := strconv.Atoi(r.URL.Query().Get("width"))
	height, _ := strconv.Atoi(r.URL.Query().Get("height"))
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "jpeg"
	}

	if width <= 0 && height <= 0 {
		writeError(w, "width or height must be specified", http.StatusBadRequest)
		return
	}

	img, err := service.DecodeImage(file)
	if err != nil {
		writeError(w, "failed to decode image: "+err.Error(), http.StatusBadRequest)
		return
	}

	resized := service.Resize(img, width, height)

	buf := new(bytes.Buffer)
	if format == "png" {
		png.Encode(buf, resized)
	} else {
		jpeg.Encode(buf, resized, &jpeg.Options{Quality: 85})
	}

	writeImage(w, buf, format)
}

// Grayscale converts image to grayscale
func Grayscale(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, "image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "jpeg"
	}

	img, err := service.DecodeImage(file)
	if err != nil {
		writeError(w, "failed to decode image: "+err.Error(), http.StatusBadRequest)
		return
	}

	gray := service.Grayscale(img)

	buf := new(bytes.Buffer)
	if format == "png" {
		png.Encode(buf, gray)
	} else {
		jpeg.Encode(buf, gray, &jpeg.Options{Quality: 85})
	}

	writeImage(w, buf, format)
}

// Rotate rotates the image by specified degrees
// Query params: angle (90, 180, 270), format (jpeg/png)
func Rotate(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, "image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	angle, _ := strconv.Atoi(r.URL.Query().Get("angle"))
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "jpeg"
	}

	if angle != 90 && angle != 180 && angle != 270 {
		writeError(w, "angle must be 90, 180, or 270", http.StatusBadRequest)
		return
	}

	img, err := service.DecodeImage(file)
	if err != nil {
		writeError(w, "failed to decode image: "+err.Error(), http.StatusBadRequest)
		return
	}

	rotated := service.Rotate(img, angle)

	buf := new(bytes.Buffer)
	if format == "png" {
		png.Encode(buf, rotated)
	} else {
		jpeg.Encode(buf, rotated, &jpeg.Options{Quality: 85})
	}

	writeImage(w, buf, format)
}
