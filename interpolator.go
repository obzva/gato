package gato

import (
	"errors"
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
)

var (
	ErrBilinearSrcImageTooSmall = errors.New("source image is too small: width < 2 or height < 2")
)

type interpolator interface {
	interpolate(src, dst *image.RGBA) error
}

func newInterpolator(method string) interpolator {
	var itp interpolator
	switch method {
	case NearestNeighbor:
		itp = &nearestNeighbor{}
	case Bilinear:
		itp = &bilinear{}
	case Bicubic:
		itp = &bicubic{}
	}
	return itp
}

type nearestNeighbor struct{}

func (n *nearestNeighbor) interpolate(src, dst *image.RGBA) error {
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()
	dstW := dst.Bounds().Dx()
	dstH := dst.Bounds().Dy()

	scaleX := getScale(srcW, dstW)
	scaleY := getScale(srcH, dstH)

	numGoroutines := runtime.NumCPU()
	total := dstW * dstH
	chunkSize := total / numGoroutines

	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for ; start < end; start++ {
				x := start % dstW
				y := start / dstW
				transX := int(math.Floor(float64(x) / scaleX))
				transY := int(math.Floor(float64(y) / scaleY))
				dst.Set(x, y, src.At(transX, transY))
			}
		}(i*chunkSize, (i+1)*chunkSize)
	}

	wg.Wait()

	return nil
}

type bilinear struct{}

// calculates the weighted average of two points(nV and nV+1) for each color channel (RGBA) about v
// pR, pG, pB, pA: color values at two points (index 0: color value of nV, index 1: color value of nV+1)
// nV: largest integer value no larger than v
func (bl *bilinear) internalDivision(pR, pG, pB, pA *[2]float64, nV, v float64) (r float64, g float64, b float64, a float64) {
	r = (nV+1-v)*float64(pR[0]) + (v-nV)*float64(pR[1])
	g = (nV+1-v)*float64(pG[0]) + (v-nV)*float64(pG[1])
	b = (nV+1-v)*float64(pB[0]) + (v-nV)*float64(pB[1])
	a = (nV+1-v)*float64(pA[0]) + (v-nV)*float64(pA[1])

	return r, g, b, a
}

func (bl *bilinear) interpolate(src, dst *image.RGBA) error {
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()
	if srcW < 2 || srcH < 2 {
		return ErrBilinearSrcImageTooSmall
	}
	dstW := dst.Bounds().Dx()
	dstH := dst.Bounds().Dy()

	scaleX := getScale(srcW, dstW)
	scaleY := getScale(srcH, dstH)

	offsetX := getOffset(scaleX)
	offsetY := getOffset(scaleY)

	numGoroutines := runtime.NumCPU()
	total := dstW * dstH
	chunkSize := total / numGoroutines

	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for ; start < end; start++ {
				x := start % dstW
				y := start / dstW
				transX := float64(x)/scaleX - offsetX
				transY := float64(y)/scaleY - offsetY

				// boundary check for edge points
				edgeX := transX < 0 || transX > float64(srcW-1)
				edgeY := transY < 0 || transY > float64(srcH-1)

				// meaning of prefix
				// n: nearest (largest integer value no larger than ...)
				// l: left
				// r: right
				// t: top
				// b: bottom
				// i: interpolated

				var iColor color.RGBA

				// use just one nearest surrounding point
				if edgeX && edgeY {
					var nX int
					var nY int

					if transX < 0 {
						nX = 0
					} else {
						nX = srcW - 1
					}

					if transY < 0 {
						nY = 0
					} else {
						nY = srcH - 1
					}

					iColor = src.RGBAAt(nX, nY)
				} else if edgeX { // use two surrounding points (only y-axis)
					var nX float64
					if transX < 0 {
						nX = 0
					} else {
						nX = float64(srcW - 1)
					}

					nY := math.Floor(transY)

					// color values at two points (nX, nY) and (nX, nY+1) for each color channel (RGBA)
					// index 0: color values at (nX, nY)
					// index 1: color values at (nX, nY+1)
					var pR, pG, pB, pA [2]float64

					for i := range 2 {
						pRGBA := src.RGBAAt(int(nX), int(nY)+i)
						pR[i] = float64(pRGBA.R)
						pG[i] = float64(pRGBA.G)
						pB[i] = float64(pRGBA.B)
						pA[i] = float64(pRGBA.A)
					}

					iR, iG, iB, iA := bl.internalDivision(&pR, &pG, &pB, &pA, nY, transY)

					iColor = color.RGBA{clamp(iR), clamp(iG), clamp(iB), clamp(iA)}
				} else if edgeY { // use two surrounding points (only x-axis)
					var nY float64

					if transY < 0 {
						nY = 0
					} else {
						nY = float64(srcH - 1)
					}

					nX := math.Floor(transX)

					// color values at two points (nX, nY) and (nX+1, nY) for each color channel (RGBA)
					// index 0: color values at (nX, nY)
					// index 1: color values at (nX+1, nY)
					var pR, pG, pB, pA [2]float64

					for i := range 2 {
						pRGBA := src.RGBAAt(int(nX)+i, int(nY))
						pR[i] = float64(pRGBA.R)
						pG[i] = float64(pRGBA.G)
						pB[i] = float64(pRGBA.B)
						pA[i] = float64(pRGBA.A)
					}

					iR, iG, iB, iA := bl.internalDivision(&pR, &pG, &pB, &pA, nX, transX)

					iColor = color.RGBA{clamp(iR), clamp(iG), clamp(iB), clamp(iA)}
				} else { // use four surrounding points (both x-axis and y-axis)
					nX := math.Floor(transX)
					nY := math.Floor(transY)

					// color values at four points (nX, nY), (nX+1, nY), (nX, nY+1) and (nX+1, nY+1) for each color channel (RGBA)
					// index [0][0]: color values at (nX, nY)
					// index [0][1]: color values at (nX+1, nY)
					// index [1][0]: color values at (nX, nY+1)
					// index [1][1]: color values at (nX+1, nY+1)
					var pR, pG, pB, pA [2][2]float64

					// temporarily saved color values got from internal division on x-axis
					// index 0: values got from internal division on y=nY
					// index 1: values got from internal division on y=nY+1
					var tmpR, tmpG, tmpB, tmpA [2]float64

					for i := range 2 {
						for j := range 2 {
							pRGBA := src.RGBAAt(int(nX)+j, int(nY)+i)
							pR[i][j] = float64(pRGBA.R)
							pG[i][j] = float64(pRGBA.G)
							pB[i][j] = float64(pRGBA.B)
							pA[i][j] = float64(pRGBA.A)
						}
						tmpR[i], tmpG[i], tmpB[i], tmpA[i] = bl.internalDivision(&pR[i], &pG[i], &pB[i], &pA[i], nX, transX)
					}

					iR, iG, iB, iA := bl.internalDivision(&tmpR, &tmpG, &tmpB, &tmpA, nY, transY)

					iColor = color.RGBA{clamp(iR), clamp(iG), clamp(iB), clamp(iA)}
				}
				dst.Set(x, y, iColor)
			}
		}(i*chunkSize, (i+1)*chunkSize)
	}

	wg.Wait()

	return nil
}

type bicubic struct{}

func (b *bicubic) interpolate(src, dst *image.RGBA) error {
	return nil
}

// return k s.t. a*k = b
func getScale(a, b int) (k float64) {
	return float64(b) / float64(a)
}

// return offset which is used to transform coordinates from output space to input space
func getOffset(scale float64) float64 {
	return (scale - 1) / (2 * scale)
}

// clamp returns the uint8value of v clamped to the range [0, 255]
func clamp(v float64) uint8 {
	if v > 255 { // overshoot
		return 255
	} else if v < 0 { // undershoot
		return 0
	} else {
		return uint8(math.Round(v))
	}
}

// initialize Interpolator
// available methods are
//   - nearestneighbor
//   - bilinear
//   - bicubic

// type NearestNeighbor struct {
// 	input, output *image.NRGBA
// }

// // returns (x-axis scale, y-axis scale) = ((output width / input width), (output height / input height))
// func (nn *NearestNeighbor) getScale() (scaleX float64, scaleY float64) {
// 	iW := nn.input.Bounds().Dx()
// 	iH := nn.input.Bounds().Dy()

// 	oW := nn.output.Bounds().Dx()
// 	oH := nn.output.Bounds().Dy()

// 	return float64(oW) / float64(iW), float64(oH) / float64(iH)
// }

// // converts coordinates from output space to input space
// func (nn *NearestNeighbor) transformCoords(x, y int) (tX float64, tY float64) {
// 	scaleX, scaleY := nn.getScale()

// 	return float64(x) / scaleX, float64(y) / scaleY
// }

// func (nn *NearestNeighbor) operate(start, end int) {
// 	oW := nn.output.Bounds().Dx()

// 	for ; start < end; start++ {
// 		x := start % oW
// 		y := start / oW

// 		tX, tY := nn.transformCoords(x, y)

// 		nn.output.Set(x, y, nn.input.At(int(tX), int(tY)))
// 	}
// }

// func (nn *NearestNeighbor) Interpolate(concurrency bool) *image.NRGBA {
// 	funcName := "Nearest neighbor"
// 	if concurrency {
// 		funcName += " with concurrency"
// 	}
// 	defer timeTrack(time.Now(), funcName)

// 	oW := nn.output.Bounds().Dx()
// 	oH := nn.output.Bounds().Dy()

// 	if concurrency {
// 		numCPU := runtime.NumCPU()
// 		c := make(chan int, numCPU)

// 		for i := range numCPU {
// 			go func() {
// 				nn.operate(i*oW*oH/numCPU, (i+1)*oW*oH/numCPU)
// 				c <- 1
// 			}()
// 		}
// 		// drain the channel
// 		for i := 0; i < numCPU; i++ {
// 			<-c
// 		}
// 		// all done
// 	} else {
// 		nn.operate(0, oW*oH)
// 	}

// 	return nn.output
// }

// type Bilinear struct {
// 	input, output *image.NRGBA
// }

// // returns (x-axis scale, y-axis scale) = ((output width / input width), (output height / input height))
// func (bl *Bilinear) getScale() (scaleX float64, scaleY float64) {
// 	iW := bl.input.Bounds().Dx()
// 	iH := bl.input.Bounds().Dy()

// 	oW := bl.output.Bounds().Dx()
// 	oH := bl.output.Bounds().Dy()

// 	return float64(oW) / float64(iW), float64(oH) / float64(iH)
// }

// // converts coordinates from output space to input space
// func (bl *Bilinear) transformCoords(x, y int) (tX float64, tY float64) {
// 	scaleX, scaleY := bl.getScale()

// 	offsetX := getOffset(scaleX)
// 	offsetY := getOffset(scaleY)

// 	return float64(x)/scaleX - offsetX, float64(y)/scaleY - offsetY
// }

// // calculates the weighted average of two points(nV and nV+1) for each color channel (RGBA) about v
// // pR, pG, pB, pA: color values at two points (index 0: color value of nV, index 1: color value of nV+1)
// // nV: largest integer value no larger than v
// func (bl *Bilinear) internalDivision(pR, pG, pB, pA *[2]float64, nV, v float64) (r float64, g float64, b float64, a float64) {
// 	r = (nV+1-v)*float64(pR[0]) + (v-nV)*float64(pR[1])
// 	g = (nV+1-v)*float64(pG[0]) + (v-nV)*float64(pG[1])
// 	b = (nV+1-v)*float64(pB[0]) + (v-nV)*float64(pB[1])
// 	a = (nV+1-v)*float64(pA[0]) + (v-nV)*float64(pA[1])

// 	return r, g, b, a
// }

// func (bl *Bilinear) operate(start, end int) {
// 	iW := bl.input.Bounds().Dx()
// 	iH := bl.input.Bounds().Dy()

// 	oW := bl.output.Bounds().Dx()

// 	for ; start < end; start++ {
// 		x := start % oW
// 		y := start / oW

// 		// transformed x and y
// 		tX, tY := bl.transformCoords(x, y)

// 		// boundary check
// 		outX := tX < 0 || tX > float64(iW-1)
// 		outY := tY < 0 || tY > float64(iH-1)

// 		var iC color.NRGBA

// 		// meaning of prefix
// 		// n: nearest (largest integer value no larger than ...)
// 		// l: left
// 		// r: right
// 		// t: top
// 		// b: bottom

// 		// use just one nearest surrounding point
// 		if outX && outY {
// 			var nX int
// 			var nY int

// 			if tX < 0 {
// 				nX = 0
// 			} else {
// 				nX = iW - 1
// 			}

// 			if tY < 0 {
// 				nY = 0
// 			} else {
// 				nY = iH - 1
// 			}

// 			iC = bl.input.NRGBAAt(nX, nY)
// 		} else if outX { // use two surrounding points (only y-axis)
// 			var nX float64
// 			if tX < 0 {
// 				nX = 0
// 			} else {
// 				nX = float64(iW - 1)
// 			}

// 			nY := math.Floor(tY)

// 			// color values at two points (nX, nY) and (nX, nY+1) for each color channel (RGBA)
// 			// index 0: color values at (nX, nY)
// 			// index 1: color values at (nX, nY+1)
// 			var pR, pG, pB, pA [2]float64

// 			for i := range 2 {
// 				pRGBA := bl.input.NRGBAAt(int(nX), int(nY)+i)
// 				pR[i] = float64(pRGBA.R)
// 				pG[i] = float64(pRGBA.G)
// 				pB[i] = float64(pRGBA.B)
// 				pA[i] = float64(pRGBA.A)
// 			}

// 			iR, iG, iB, iA := bl.internalDivision(&pR, &pG, &pB, &pA, nY, tY)

// 			iC = color.NRGBA{clamp(iR), clamp(iG), clamp(iB), clamp(iA)}
// 		} else if outY { // use two surrounding points (only x-axis)
// 			var nY float64

// 			if tY < 0 {
// 				nY = 0
// 			} else {
// 				nY = float64(iH - 1)
// 			}

// 			nX := math.Floor(tX)

// 			// color values at two points (nX, nY) and (nX+1, nY) for each color channel (RGBA)
// 			// index 0: color values at (nX, nY)
// 			// index 1: color values at (nX+1, nY)
// 			var pR, pG, pB, pA [2]float64

// 			for i := range 2 {
// 				pRGBA := bl.input.NRGBAAt(int(nX)+i, int(nY))
// 				pR[i] = float64(pRGBA.R)
// 				pG[i] = float64(pRGBA.G)
// 				pB[i] = float64(pRGBA.B)
// 				pA[i] = float64(pRGBA.A)
// 			}

// 			iR, iG, iB, iA := bl.internalDivision(&pR, &pG, &pB, &pA, nX, tX)

// 			iC = color.NRGBA{clamp(iR), clamp(iG), clamp(iB), clamp(iA)}
// 		} else { // use four surrounding points (both x-axis and y-axis)
// 			nX := math.Floor(tX)
// 			nY := math.Floor(tY)

// 			// color values at four points (nX, nY), (nX+1, nY), (nX, nY+1) and (nX+1, nY+1) for each color channel (RGBA)
// 			// index [0][0]: color values at (nX, nY)
// 			// index [0][1]: color values at (nX+1, nY)
// 			// index [1][0]: color values at (nX, nY+1)
// 			// index [1][1]: color values at (nX+1, nY+1)
// 			var pR, pG, pB, pA [2][2]float64

// 			// temporarily saved color values got from internal division on x-axis
// 			// index 0: values got from internal division on y=nY
// 			// index 1: values got from internal division on y=nY+1
// 			var tmpR, tmpG, tmpB, tmpA [2]float64

// 			for i := range 2 {
// 				for j := range 2 {
// 					pRGBA := bl.input.NRGBAAt(int(nX)+j, int(nY)+i)
// 					pR[i][j] = float64(pRGBA.R)
// 					pG[i][j] = float64(pRGBA.G)
// 					pB[i][j] = float64(pRGBA.B)
// 					pA[i][j] = float64(pRGBA.A)
// 				}
// 				tmpR[i], tmpG[i], tmpB[i], tmpA[i] = bl.internalDivision(&pR[i], &pG[i], &pB[i], &pA[i], nX, tX)
// 			}

// 			iR, iG, iB, iA := bl.internalDivision(&tmpR, &tmpG, &tmpB, &tmpA, nY, tY)

// 			iC = color.NRGBA{clamp(iR), clamp(iG), clamp(iB), clamp(iA)}
// 		}
// 		bl.output.Set(x, y, iC)
// 	}
// }

// func (bl *Bilinear) Interpolate(concurrency bool) *image.NRGBA {
// 	funcName := "Bilinear"
// 	if concurrency {
// 		funcName += " with concurrency"
// 	}
// 	defer timeTrack(time.Now(), funcName)

// 	oW := bl.output.Bounds().Dx()
// 	oH := bl.output.Bounds().Dy()

// 	if concurrency {
// 		numCPU := runtime.NumCPU()
// 		c := make(chan int, numCPU)

// 		for i := range numCPU {
// 			go func() {
// 				bl.operate(i*oW*oH/numCPU, (i+1)*oW*oH/numCPU)
// 				c <- 1
// 			}()
// 		}
// 		// drain the channel
// 		for i := 0; i < numCPU; i++ {
// 			<-c
// 		}
// 		// all done
// 	} else {
// 		bl.operate(0, oW*oH)
// 	}

// 	return bl.output
// }

// type Bicubic struct {
// 	input, output *image.NRGBA
// }

// // returns (x-axis scale, y-axis scale) = ((output width / input width), (output height / input height))
// func (bc *Bicubic) getScale() (scaleX float64, scaleY float64) {
// 	iW := bc.input.Bounds().Dx()
// 	iH := bc.input.Bounds().Dy()

// 	oW := bc.output.Bounds().Dx()
// 	oH := bc.output.Bounds().Dy()

// 	return float64(oW) / float64(iW), float64(oH) / float64(iH)
// }

// // converts coordinates from output space to input space
// func (bc *Bicubic) transformCoords(x, y int) (tX float64, tY float64) {
// 	scaleX, scaleY := bc.getScale()

// 	offsetX := getOffset(scaleX)
// 	offsetY := getOffset(scaleY)

// 	return float64(x)/scaleX - offsetX, float64(y)/scaleY - offsetY
// }

// // interpolates a value f(v) that function f(t) takes at ordinate t=v
// // for more detail of formula, please refer to https://en.wikipedia.org/wiki/Cubic_Hermite_spline#Interpolation_on_the_unit_interval_with_matched_derivatives_at_endpoints
// // u: fractional part of v
// // p: color values at four points(p_n-1, p_n, p_n+1, p_n+2) for each color channel (RGBA)
// //   - index 0: color values at p_n-1
// //   - index 1: color values at p_n
// //   - index 2: color values at p_n+1
// //   - index 3: color values at p_n+2
// func (bc *Bicubic) catmullRomSpline(u float64, p *[4]float64) float64 {
// 	u2 := u * u
// 	u3 := u2 * u

// 	term1 := (-p[0] + 3*p[1] - 3*p[2] + p[3]) * u3
// 	term2 := (2*p[0] - 5*p[1] + 4*p[2] - p[3]) * u2
// 	term3 := (-p[0] + p[2]) * u
// 	term4 := 2 * p[1]

// 	return 0.5 * (term1 + term2 + term3 + term4)
// }

// func (bc *Bicubic) operate(start, end int) {
// 	iW := bc.input.Bounds().Dx()
// 	iH := bc.input.Bounds().Dy()

// 	oW := bc.output.Bounds().Dx()

// 	for ; start < end; start++ {
// 		x := start % oW
// 		y := start / oW

// 		// transformed x and y
// 		tX, tY := bc.transformCoords(x, y)

// 		// boundary check
// 		outX := tX < 1 || tX > float64(iW-2)
// 		outY := tY < 1 || tY > float64(iH-2)

// 		var iC color.NRGBA

// 		// use just one nearest surrounding point
// 		if outX && outY {
// 			var nX int
// 			var nY int

// 			if tX < 0.5 {
// 				nX = 0
// 			} else if tX < 1 {
// 				nX = 1
// 			} else if tX <= float64(iW)-1.5 {
// 				nX = iW - 2
// 			} else {
// 				nX = iW - 1
// 			}

// 			if tY < 0.5 {
// 				nY = 0
// 			} else if tY < 1 {
// 				nY = 1
// 			} else if tY <= float64(iH)-1.5 {
// 				nY = iH - 2
// 			} else {
// 				nY = iH - 1
// 			}

// 			iC = bc.input.NRGBAAt(nX, nY)
// 		} else if outX { // use only y-axis
// 			var nX int

// 			if tX < 0.5 {
// 				nX = 0
// 			} else if tX < 1 {
// 				nX = 1
// 			} else if tX <= float64(iW)-1.5 {
// 				nX = iW - 2
// 			} else {
// 				nX = iW - 1
// 			}

// 			floorY := math.Floor(tY)
// 			fractionY := tY - floorY

// 			intY := int(floorY)

// 			var pR, pG, pB, pA [4]float64

// 			for i := range 4 {
// 				pR[i] = float64(bc.input.NRGBAAt(nX, intY-1+i).R)
// 				pG[i] = float64(bc.input.NRGBAAt(nX, intY-1+i).G)
// 				pB[i] = float64(bc.input.NRGBAAt(nX, intY-1+i).B)
// 				pA[i] = float64(bc.input.NRGBAAt(nX, intY-1+i).A)
// 			}

// 			iR := clamp(bc.catmullRomSpline(fractionY, &pR))
// 			iG := clamp(bc.catmullRomSpline(fractionY, &pG))
// 			iB := clamp(bc.catmullRomSpline(fractionY, &pB))
// 			iA := clamp(bc.catmullRomSpline(fractionY, &pA))

// 			iC = color.NRGBA{iR, iG, iB, iA}
// 		} else if outY { // use only x-axis
// 			var nY int

// 			if tY < 0.5 {
// 				nY = 0
// 			} else if tY < 1 {
// 				nY = 1
// 			} else if tY <= float64(iH)-1.5 {
// 				nY = iH - 2
// 			} else {
// 				nY = iH - 1
// 			}

// 			floorX := math.Floor(tX)
// 			fractionX := tX - floorX

// 			intX := int(floorX)

// 			var pR, pG, pB, pA [4]float64

// 			for i := range 4 {
// 				pR[i] = float64(bc.input.NRGBAAt(intX-1+i, nY).R)
// 				pG[i] = float64(bc.input.NRGBAAt(intX-1+i, nY).G)
// 				pB[i] = float64(bc.input.NRGBAAt(intX-1+i, nY).B)
// 				pA[i] = float64(bc.input.NRGBAAt(intX-1+i, nY).A)
// 			}

// 			iR := clamp(bc.catmullRomSpline(fractionX, &pR))
// 			iG := clamp(bc.catmullRomSpline(fractionX, &pG))
// 			iB := clamp(bc.catmullRomSpline(fractionX, &pB))
// 			iA := clamp(bc.catmullRomSpline(fractionX, &pA))

// 			iC = color.NRGBA{iR, iG, iB, iA}
// 		} else { // use both two axes, x first y later
// 			floorX := math.Floor(tX)
// 			fractionX := tX - floorX

// 			intX := int(floorX)

// 			floorY := math.Floor(tY)
// 			fractionY := tY - floorY

// 			intY := int(floorY)

// 			var tmpR, tmpG, tmpB, tmpA [4]float64
// 			var pR, pG, pB, pA [4][4]float64

// 			for i := range 4 {
// 				for j := range 4 {
// 					pR[i][j] = float64(bc.input.NRGBAAt(intX-1+j, intY-1+i).R)
// 					pG[i][j] = float64(bc.input.NRGBAAt(intX-1+j, intY-1+i).G)
// 					pB[i][j] = float64(bc.input.NRGBAAt(intX-1+j, intY-1+i).B)
// 					pA[i][j] = float64(bc.input.NRGBAAt(intX-1+j, intY-1+i).A)
// 				}

// 				tmpR[i] = bc.catmullRomSpline(fractionX, &pR[i])
// 				tmpG[i] = bc.catmullRomSpline(fractionX, &pG[i])
// 				tmpB[i] = bc.catmullRomSpline(fractionX, &pB[i])
// 				tmpA[i] = bc.catmullRomSpline(fractionX, &pA[i])
// 			}

// 			iR := clamp(bc.catmullRomSpline(fractionY, &tmpR))
// 			iG := clamp(bc.catmullRomSpline(fractionY, &tmpG))
// 			iB := clamp(bc.catmullRomSpline(fractionY, &tmpB))
// 			iA := clamp(bc.catmullRomSpline(fractionY, &tmpA))

// 			iC = color.NRGBA{iR, iG, iB, iA}
// 		}
// 		bc.output.Set(x, y, iC)
// 	}
// }

// func (bc *Bicubic) Interpolate(concurrency bool) *image.NRGBA {
// 	funcName := "Bicubic"
// 	if concurrency {
// 		funcName += " with concurrency"
// 	}
// 	defer timeTrack(time.Now(), funcName)

// 	oW := bc.output.Bounds().Dx()
// 	oH := bc.output.Bounds().Dy()

// 	if concurrency {
// 		numCPU := runtime.NumCPU()
// 		c := make(chan int, numCPU)

// 		for i := range numCPU {
// 			go func() {
// 				bc.operate(i*oW*oH/numCPU, (i+1)*oW*oH/numCPU)
// 				c <- 1
// 			}()
// 		}
// 		// drain the channel
// 		for i := 0; i < numCPU; i++ {
// 			<-c
// 		}
// 		// all done
// 	} else {
// 		bc.operate(0, oW*oH)
// 	}

// 	return bc.output
// }

// // helpers
// func timeTrack(start time.Time, funcName string) {
// 	elapsed := time.Since(start)
// 	fmt.Printf("%s interpolation took %v to run\n", funcName, elapsed)
// }

// func getOffset(scale float64) float64 {
// 	return (scale - 1) / (2 * scale)
// }

// func clamp(v float64) uint8 {
// 	if v > 255 { // overshoot
// 		return 255
// 	} else if v < 0 { // undershoot
// 		return 0
// 	} else {
// 		return uint8(math.Round(v))
// 	}
// }
