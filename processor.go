package gato

import (
	"errors"
	"image"
	"math"
)

const (
	NearestNeighbor = "nearest-neighbor"
	Bilinear        = "bilinear"
	Bicubic         = "bicubic"
)

var (
	ErrInvalidDimension     = errors.New("invalid dimension: one of the dimension is not set or set to 0")
	ErrInvalidInterpolation = errors.New("invalid interpolation method: only nearest-neighbor, bilinear, and bicubic are available")
)

// Instruction is a struct that contains the instruction for the processor.
type Instruction struct {
	Width         int
	Height        int
	Interpolation string
}

// Processor is a struct that contains the instruction and related helpers
type Processor struct {
	Instruction
	Interpolator interpolator
}

// return the processed image following the instructions
func (p *Processor) Process(d *Data) (*image.RGBA, error) {
	// setting dimensions
	w := p.Width
	h := p.Height
	if w == 0 {
		srcW := d.Image.Bounds().Dx()
		srcH := d.Image.Bounds().Dy()
		scale := float64(h) / float64(srcH)
		w = int(math.Round(scale * float64(srcW)))
	}
	if h == 0 {
		srcW := d.Image.Bounds().Dx()
		srcH := d.Image.Bounds().Dy()
		scale := float64(w) / float64(srcW)
		h = int(math.Round(scale * float64(srcH)))
	}

	rect := image.Rect(0, 0, w, h)
	rgba := image.NewRGBA(rect)

	err := p.Interpolator.interpolate(d.Image, rgba)
	if err != nil {
		return nil, err
	}

	return rgba, nil
}

// NewProcessor creates a new Processor instance from an Instruction instance.
// If the Instruction.Width and Instruction.Height are not set, it returns an error ErrInvalidDimension.
// It also creates a new Interpolator instance from the Interpolation instruction. If Instruction.Interpolation is not set, it defaults to Bilinear.
func NewProcessor(i Instruction) (*Processor, error) {
	if i.Width == 0 && i.Height == 0 {
		return nil, ErrInvalidDimension
	}

	switch i.Interpolation {
	case "":
		i.Interpolation = Bilinear
	case NearestNeighbor, Bilinear, Bicubic:
		// do nothing
	default:
		return nil, ErrInvalidInterpolation
	}

	itp := newInterpolator(i.Interpolation)

	return &Processor{
		Instruction:  i,
		Interpolator: itp,
	}, nil
}
