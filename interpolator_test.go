package gato

import (
	"image"
	"image/color"
	"math"
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
	_ = nn.interpolate(src, dst)

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

func TestBilinear(t *testing.T) {
	t.Run("test internalDivision method", func(t *testing.T) {
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

func TestCatmullRomSpline(t *testing.T) {

}

func TestBiCubic(t *testing.T) {
	t.Run("test catmullRomSpline method", func(t *testing.T) {
		bc := &bicubic{}
		u := 0.5
		p := [4]float64{0, 100, 200, 150}
		want := 159.375

		got := bc.catmullRomSpline(u, &p)
		if math.Abs(got-want) > 0.001 {
			t.Errorf("catmullRomSpline() = %v, want %v", got, want)
		}
	})

	t.Run("return error when source image is too small", func(t *testing.T) {
		bc := &bicubic{}
		src := image.NewRGBA(image.Rect(0, 0, 3, 3))
		dst := image.NewRGBA(image.Rect(0, 0, 10, 10))
		err := bc.interpolate(src, dst)
		if err != ErrBicubicSrcImageTooSmall {
			t.Errorf("got %v, want %v", err, ErrBicubicSrcImageTooSmall)
		}
	})
}
