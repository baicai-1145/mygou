// Command mkicon draws the app icon: a blue rounded square with a white
// music note, the same mark as the menu bar's.
//
//	go run ./cmd/mkicon -out build/icon.png
package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

func main() {
	out := flag.String("out", "build/icon.png", "output PNG")
	flag.Parse()

	const size = 1024
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	// Rounded square, with the gradient of the player's primary color.
	const radius = 225.0
	top := color.NRGBA{R: 84, G: 122, B: 255, A: 255}
	bottom := color.NRGBA{R: 26, G: 54, B: 180, A: 255}
	inCorner := func(x, y float64) bool {
		cx := math.Min(math.Max(x, radius), size-radius)
		cy := math.Min(math.Max(y, radius), size-radius)
		dx, dy := x-cx, y-cy
		return dx*dx+dy*dy <= radius*radius
	}
	for y := 0; y < size; y++ {
		t := float64(y) / (size - 1)
		// Ease the gradient so the light stays near the top.
		t = math.Pow(t, 1.25)
		row := color.NRGBA{
			R: uint8(float64(top.R) + (float64(bottom.R)-float64(top.R))*t),
			G: uint8(float64(top.G) + (float64(bottom.G)-float64(top.G))*t),
			B: uint8(float64(top.B) + (float64(bottom.B)-float64(top.B))*t),
			A: 255,
		}
		for x := 0; x < size; x++ {
			if inCorner(float64(x), float64(y)) {
				img.SetNRGBA(x, y, row)
			}
		}
	}

	white := color.NRGBA{R: 255, G: 255, B: 255, A: 235}
	disc := func(cx, cy, r float64) {
		for y := int(cy - r - 2); y <= int(cy+r+2); y++ {
			for x := int(cx - r - 2); x <= int(cx+r+2); x++ {
				dx, dy := float64(x)-cx, float64(y)-cy
				if dx*dx+dy*dy <= r*r {
					img.SetNRGBA(x, y, white)
				}
			}
		}
	}
	rect := func(x0, y0, x1, y1 float64) {
		for y := int(y0); y <= int(y1); y++ {
			for x := int(x0); x <= int(x1); x++ {
				img.SetNRGBA(x, y, white)
			}
		}
	}

	// A beamed pair of notes: heads at the baseline, stems to the beam.
	disc(395, 700, 92)
	disc(700, 640, 92)
	rect(462, 330, 500, 700)
	rect(767, 270, 805, 640)
	// The beam, thicker on the left, as a slanted quad.
	for x := 462; x <= 805; x++ {
		t := float64(x-462) / float64(805-462)
		topY := 330 - t*60
		bottomY := topY + 96
		for y := int(topY); y <= int(bottomY); y++ {
			img.SetNRGBA(x, y, white)
		}
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}
