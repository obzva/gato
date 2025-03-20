# gato

**gato** is a simple package for image processing operation written in Go

## Features

- Supporting JPG/JPEG and PNG for input image
- Resize
  - For resizing, there are three interpolation methods available:
    - Nearest Neighbor
    - Bilinear
    - Bicubic
      - Using [Catmull Rom Spline](https://en.wikipedia.org/wiki/Cubic_Hermite_spline#Interpolation_on_the_unit_interval_with_matched_derivatives_at_endpoints)

## Usage

Here's a quick example of how to use **gato**:

```go
package main

import (
	"fmt"
	"image/jpeg"
	"log"
	"os"

	"github.com/obzva/gato-cli/gato"
)

func main() {
  fileName := "path/your-image.jpg"

  // read image
	img, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer img.Close()

  // create a new Data struct
  // pass fileName and io.Reader interface
	data, err := gato.NewData(fileName, img)
	if err != nil {
		log.Fatal(err)
	}

  // create a new Processor struct
  // pass your instructions with Instruction struct
	prc, err := gato.NewProcessor(gato.Instruction{
		Width:         1500,
		Interpolation: gato.NearestNeighbor,
	})
	if err != nil {
		log.Fatal(err)
	}

  // process the image
	res, err := prc.Process(data)
	if err != nil {
		log.Fatal(err)
	}

  // ...
}
```
