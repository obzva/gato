package gato

import (
	"testing"
)

func TestProcessor(t *testing.T) {
	t.Run("save width and height correctly", func(t *testing.T) {
		w := 100
		h := 120
		i := Instruction{Width: w, Height: h}
		p, _ := NewProcessor(i)
		assertInt(t, p.Width, w)
		assertInt(t, p.Height, h)
	})

	t.Run("return error when both of the dimension is not set or set to 0", func(t *testing.T) {
		i := Instruction{}
		_, got := NewProcessor(i)
		want := ErrInvalidDimension
		assertError(t, got, want)
	})

	t.Run("create a correct interpolator", func(t *testing.T) {
		m := NearestNeighbor
		i := Instruction{Width: 100, Interpolation: m}
		p, _ := NewProcessor(i)
		if _, ok := p.interpolator.(*nearestNeighbor); !ok {
			t.Errorf("got %T, want %T", p.interpolator, &nearestNeighbor{})
		}

		m = Bilinear
		i = Instruction{Width: 100, Interpolation: m}
		p, _ = NewProcessor(i)
		if _, ok := p.interpolator.(*bilinear); !ok {
			t.Errorf("got %T, want %T", p.interpolator, &bilinear{})
		}

		m = Bicubic
		i = Instruction{Width: 100, Interpolation: m}
		p, _ = NewProcessor(i)
		if _, ok := p.interpolator.(*bicubic); !ok {
			t.Errorf("got %T, want %T", p.interpolator, &bicubic{})
		}
	})

	t.Run("if omitted interpolation method, use bilinear as default", func(t *testing.T) {
		i := Instruction{Width: 100}
		p, _ := NewProcessor(i)
		if _, ok := p.interpolator.(*bilinear); !ok {
			t.Errorf("got %T, want %T", p.interpolator, &bilinear{})
		}
	})

	t.Run("return error when invalid interpolation method is provided", func(t *testing.T) {
		m := "full crimp"
		i := Instruction{Width: 100, Interpolation: m}
		_, got := NewProcessor(i)
		assertError(t, got, ErrInvalidInterpolation)
	})

	t.Run("return output image right sized", func(t *testing.T) {
		d, _ := NewData("norwich-terrier.jpg", newStubImageReader())
		w := 100
		h := 100
		i := Instruction{Width: w, Height: h}
		p, _ := NewProcessor(i)
		result, _ := p.Process(d)
		assertInt(t, result.Bounds().Dx(), w)
		assertInt(t, result.Bounds().Dy(), h)
	})

	t.Run("if width instruction is unset, keep the ratio of the original image", func(t *testing.T) {
		d, _ := NewData("norwich-terrier.jpg", newStubImageReader())
		scale := 2
		w := d.Image.Bounds().Dx() * scale
		h := d.Image.Bounds().Dy() * scale
		i := Instruction{Height: h}
		p, _ := NewProcessor(i)
		result, _ := p.Process(d)
		assertInt(t, result.Bounds().Dx(), w)
		assertInt(t, result.Bounds().Dy(), h)
	})

	t.Run("if height instruction is unset, keep the ratio of the original image", func(t *testing.T) {
		d, _ := NewData("norwich-terrier.jpg", newStubImageReader())
		scale := 2
		w := d.Image.Bounds().Dx() * scale
		h := d.Image.Bounds().Dy() * scale
		i := Instruction{Height: w}
		p, _ := NewProcessor(i)
		result, _ := p.Process(d)
		assertInt(t, result.Bounds().Dx(), w)
		assertInt(t, result.Bounds().Dy(), h)
	})
}
