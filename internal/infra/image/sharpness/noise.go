package sharpness

import (
	"image"
	"math"
)

func NoiseScore(img *image.RGBA) float64 {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var noisyPixels int
	var totalPixels int

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			c := img.RGBAAt(x, y)

			neighbors := [8][3]float64{}
			idx := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					c := img.RGBAAt(x+dx, y+dy)
					neighbors[idx] = [3]float64{float64(c.R), float64(c.G), float64(c.B)}
					idx++
				}
			}

			var rMean, gMean, bMean float64
			for _, n := range neighbors {
				rMean += n[0]
				gMean += n[1]
				bMean += n[2]
			}
			rMean /= 8
			gMean /= 8
			bMean /= 8

			var rVar, gVar, bVar float64
			for _, n := range neighbors {
				rVar += (n[0] - rMean) * (n[0] - rMean)
				gVar += (n[1] - gMean) * (n[1] - gMean)
				bVar += (n[2] - bMean) * (n[2] - bMean)
			}
			rVar /= 8
			gVar /= 8
			bVar /= 8

			rDiff := math.Abs(float64(c.R) - rMean)
			gDiff := math.Abs(float64(c.G) - gMean)
			bDiff := math.Abs(float64(c.B) - bMean)

			const diffThreshold = 20.0
			const varThreshold = 50.0

			avgDiff := (rDiff + gDiff + bDiff) / 3
			avgVar := (rVar + gVar + bVar) / 3

			if avgDiff > diffThreshold && avgVar < varThreshold {
				noisyPixels++
			}

			totalPixels++
		}
	}

	if totalPixels == 0 {
		return 0
	}

	return float64(noisyPixels) / float64(totalPixels) * 100
}
