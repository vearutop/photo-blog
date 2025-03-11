package sharpness

import (
	"image"
	"math"
	"sync"
)

const blockSize = 32

func Sobel9x9(rgbaImg *image.RGBA) float64 {
	bounds := rgbaImg.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	// Sobel 9x9.
	sobelX := [81]int{
		-4, -3, -2, -1, 0, 1, 2, 3, 4,
		-5, -4, -3, -2, 0, 2, 3, 4, 5,
		-6, -5, -4, -3, 0, 3, 4, 5, 6,
		-7, -6, -5, -4, 0, 4, 5, 6, 7,
		-8, -7, -6, -5, 0, 5, 6, 7, 8,
		-7, -6, -5, -4, 0, 4, 5, 6, 7,
		-6, -5, -4, -3, 0, 3, 4, 5, 6,
		-5, -4, -3, -2, 0, 2, 3, 4, 5,
		-4, -3, -2, -1, 0, 1, 2, 3, 4,
	}
	sobelY := [81]int{
		-4, -5, -6, -7, -8, -7, -6, -5, -4,
		-3, -4, -5, -6, -7, -6, -5, -4, -3,
		-2, -3, -4, -5, -6, -5, -4, -3, -2,
		-1, -2, -3, -4, -5, -4, -3, -2, -1,
		0, 0, 0, 0, 0, 0, 0, 0, 0,
		1, 2, 3, 4, 5, 4, 3, 2, 1,
		2, 3, 4, 5, 6, 5, 4, 3, 2,
		3, 4, 5, 6, 7, 6, 5, 4, 3,
		4, 5, 6, 7, 8, 7, 6, 5, 4,
	}

	var totalSharpness float64
	var totalBlocks int
	var mu sync.Mutex

	var wg sync.WaitGroup
	blocksPerRow := (width + blockSize - 1) / blockSize
	blocksPerCol := (height + blockSize - 1) / blockSize

	for by := 0; by < blocksPerCol; by++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for bx := 0; bx < blocksPerRow; bx++ {
				xStart := bx * blockSize
				yStart := by * blockSize

				sharpness := analyzeBlock(rgbaImg, xStart, yStart, width, height, sobelX, sobelY)
				mu.Lock()
				totalSharpness += sharpness
				totalBlocks++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if totalBlocks == 0 {
		return 0
	}
	return totalSharpness / float64(totalBlocks)
}

func analyzeBlock(img *image.RGBA, xStart, yStart, width, height int, sobelX, sobelY [81]int) float64 {
	var edgeStrength float64
	var edgePixels int

	edgeMap := make([][]bool, blockSize)
	for i := range edgeMap {
		edgeMap[i] = make([]bool, blockSize)
	}

	// 9x9 window with 4px offset.
	for y := yStart + 4; y < yStart+blockSize-4 && y < height-4; y++ {
		for x := xStart + 4; x < xStart+blockSize-4 && x < width-4; x++ {
			var rGx, rGy, gGx, gGy, bGx, bGy float64
			idx := 0
			for dy := -4; dy <= 4; dy++ {
				for dx := -4; dx <= 4; dx++ {
					offset := (y+dy)*img.Stride + (x+dx)*4
					rVal := float64(img.Pix[offset])
					gVal := float64(img.Pix[offset+1])
					bVal := float64(img.Pix[offset+2])

					rGx += rVal * float64(sobelX[idx])
					rGy += rVal * float64(sobelY[idx])
					gGx += gVal * float64(sobelX[idx])
					gGy += gVal * float64(sobelY[idx])
					bGx += bVal * float64(sobelX[idx])
					bGy += bVal * float64(sobelY[idx])
					idx++
				}
			}

			rGrad := math.Sqrt(rGx*rGx + rGy*rGy)
			gGrad := math.Sqrt(gGx*gGx + gGy*gGy)
			bGrad := math.Sqrt(bGx*bGx + bGy*bGy)
			avgGrad := (rGrad + gGrad + bGrad) / 3

			const edgeThreshold = 50.0
			if avgGrad > edgeThreshold {
				edgeMap[y-yStart][x-xStart] = true
				edgeStrength += avgGrad
				edgePixels++
			}
		}
	}

	var lineScore float64

	// Connectivity in 9x9 window (offset 4).
	for y := 4; y < blockSize-4; y++ {
		for x := 4; x < blockSize-4; x++ {
			if edgeMap[y][x] {
				neighbors := 0
				for dy := -4; dy <= 4; dy++ {
					for dx := -4; dx <= 4; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						ny, nx := y+dy, x+dx
						if ny >= 0 && ny < blockSize && nx >= 0 && nx < blockSize && edgeMap[ny][nx] {
							neighbors++
						}
					}
				}
				if neighbors >= 8 {
					lineScore += edgeStrength / float64(edgePixels)
				}
			}
		}
	}

	if edgePixels == 0 {
		return 0
	}

	return lineScore / float64(edgePixels)
}
