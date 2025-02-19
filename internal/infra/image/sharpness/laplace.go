package sharpness

import (
	"image"
	"image/color"
)

// FirstPercentile returns minimal sharpness level of 1% sharpest pixels.
// Result value is between 0 (blur) and 255 (sharpest).
func FirstPercentile(gray *image.Gray) (uint8, error) {
	sh, _, err := Custom(gray, nil, 0.01)

	return sh, err
}

// Custom collects sharpness histogram using Laplacian-8 edge-detection kernel.
//
// Non-empty resultEdge can be provided to collect edges image:
//
//	resultEdge = image.NewGray(gray.Bounds())
//
// Minimal sharpness level of top fraction is returned first, sharpness distribution is returned second.
// Sharpest level is 255.
func Custom(gray *image.Gray, resultEdge *image.Gray, topFrac float64) (uint8, [256]int64, error) {
	histogram, err := laplacianGrayK8(gray, resultEdge)
	if err != nil {
		return 0, histogram, err
	}

	size := gray.Bounds().Size()
	total := size.X * size.Y
	top := int64(topFrac * float64(total))
	sum := int64(0)
	sh := uint8(0)

	for i := 255; i >= 0; i-- {
		sum += histogram[i]

		if sum >= top {
			sh = uint8(i)

			break
		}
	}

	return sh, histogram, nil
}

func laplacianGrayK8(img *image.Gray, resultImage *image.Gray) ([256]int64, error) {
	kernel := [][]int16{
		{1, 1, 1},
		{1, -8, 1},
		{1, 1, 1},
	}

	// Kernel size.
	ks := 3
	histogram := [256]int64{}
	originalSize := img.Bounds().Size()

	for y := 0; y < originalSize.Y; y++ {
		for x := 0; x < originalSize.X; x++ {
			sum := int16(0)
			for ky := 0; ky < ks; ky++ {
				for kx := 0; kx < ks; kx++ {
					xkx := x + kx
					yky := y + ky

					if dx := xkx - originalSize.X; dx >= 0 {
						xkx = originalSize.X - dx - 1
					}

					if dy := yky - originalSize.Y; dy >= 0 {
						yky = originalSize.Y - dy - 1
					}

					pixel := img.GrayAt(xkx, yky)
					kE := kernel[kx][ky]
					sum += int16(pixel.Y) * kE
				}
			}

			sum8 := clamp(sum, 0, ^uint8(0))

			histogram[sum8]++

			if resultImage != nil {
				resultImage.Set(x, y, color.Gray{Y: sum8})
			}
		}
	}

	return histogram, nil
}

func clamp(value int16, min uint8, max uint8) uint8 {
	if value < int16(min) {
		return min
	} else if value > int16(max) {
		return max
	}
	return uint8(value)
}
