package gato

import (
	"image"
	"image/color"
	"testing"
)

func TestNearestNeighbor(t *testing.T) {
	dim := 2
	scale := 2
	colors := [][]color.RGBA{
		{color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}},
		{color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 0, 255}},
	}
	src := image.NewRGBA(image.Rect(0, 0, dim, dim))
	for y := range dim {
		for x := range dim {
			src.Set(x, y, colors[x][y])
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, dim*scale, dim*scale))
	nn := &nearestNeighbor{}
	nn.interpolate(src, dst)

	for y := range dim * scale {
		for x := range dim * scale {
			got := dst.RGBAAt(x, y)
			want := colors[x/scale][y/scale]
			if got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	}
}

func TestInternalDivision(t *testing.T) {
	t.Run("return correct values", func(t *testing.T) {
		bl := &bilinear{}
		pR := [2]float64{0, 255}
		pG := [2]float64{0, 255}
		pB := [2]float64{0, 255}
		pA := [2]float64{255, 255}
		nV := float64(0)
		v := float64(0.5)
		wantR := float64(127.5)
		wantG := float64(127.5)
		wantB := float64(127.5)
		wantA := float64(255)

		gotR, gotG, gotB, gotA := bl.internalDivision(&pR, &pG, &pB, &pA, nV, v)
		if gotR != wantR || gotG != wantG || gotB != wantB || gotA != wantA {
			t.Errorf("got (%v, %v, %v, %v), want (%v, %v, %v, %v)",
				gotR, gotG, gotB, gotA, wantR, wantG, wantB, wantA)
		}
	})

	t.Run("return error when source image is too small", func(t *testing.T) {
		bl := &bilinear{}
		src := image.NewRGBA(image.Rect(0, 0, 1, 1))
		dst := image.NewRGBA(image.Rect(0, 0, 2, 2))
		err := bl.interpolate(src, dst)
		if err != ErrBilinearSrcImageTooSmall {
			t.Errorf("got %v, want %v", err, ErrBilinearSrcImageTooSmall)
		}
	})
}

// func TestCatmullRomSpline(t *testing.T) {
// 	bc := &bicubic{}
// 	tests := []struct {
// 		name string
// 		u    float64
// 		p    [4]float64
// 		want float64
// 	}{
// 		{
// 			name: "Test middle point interpolation",
// 			u:    0.5,
// 			p:    [4]float64{0, 100, 200, 255},
// 			want: 150,
// 		},
// 		{
// 			name: "Test start point",
// 			u:    0,
// 			p:    [4]float64{0, 100, 200, 255},
// 			want: 100,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := bc.catmullRomSpline(tt.u, &tt.p)
// 			if math.Abs(got-tt.want) > 0.001 {
// 				t.Errorf("catmullRomSpline() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
