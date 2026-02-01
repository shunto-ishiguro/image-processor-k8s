package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
)

func main() {
	width, height := 800, 600

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// カラフルなグラデーション + パターンを生成
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// グラデーション
			r := uint8(255 * x / width)
			g := uint8(255 * y / height)
			b := uint8(128 + 127*math.Sin(float64(x+y)/30))

			// チェッカーパターンを重ねる
			if (x/40+y/40)%2 == 0 {
				r = uint8(float64(r) * 0.8)
				g = uint8(float64(g) * 0.8)
				b = uint8(float64(b) * 0.8)
			}

			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// ファイル保存
	filename := "testdata/sample.jpg"
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	file, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 90}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated: %s (800x600)\n", filename)
}
