package gato

import (
	"errors"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"regexp"
)

var (
	ErrInvalidFileName = errors.New("invalid file name")
	ErrInvalidFormat   = errors.New("invalid format: only jpg/jpeg and png formats are supported")
)

type Data struct {
	Name   string
	Format string
	Image  *image.RGBA
}

func NewData(fileName string, r io.ReadCloser) (*Data, error) {
	defer r.Close()

	// extract name and format from fileName
	re, err := regexp.Compile(`^(.+)\.([^.]+)$`)
	if err != nil {
		return nil, err
	}
	matches := re.FindStringSubmatch(fileName)
	if len(matches) != 3 {
		return nil, ErrInvalidFileName
	}

	imgName := matches[1]
	format := matches[2]
	if format == "jpg" {
		format = "jpeg"
	}
	if format != "jpeg" && format != "png" {
		return nil, ErrInvalidFormat
	}

	// decode []byte to *image.RGBA
	dec, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	rect := dec.Bounds()
	w, h := rect.Size().X, rect.Size().Y
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rect, dec, rect.Min, draw.Src)

	data := &Data{
		Name:   imgName,
		Format: format,
		Image:  rgba,
	}

	return data, nil
}
