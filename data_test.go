package gato

import (
	"image"
	"testing"
)

func TestData(t *testing.T) {
	t.Run("save name field correctly", func(t *testing.T) {
		data, _ := NewData("norwich-terrier.jpg", newStubImageReader())
		got := data.Name
		want := "norwich-terrier"
		assertString(t, got, want)
	})

	t.Run("save jpeg format correctly", func(t *testing.T) {
		data, _ := NewData("norwich-terrier.jpeg", newStubImageReader())
		got := data.Format
		want := "jpeg"
		assertString(t, got, want)
	})

	t.Run("save jpeg format correctly when jpg format is passed", func(t *testing.T) {
		data, _ := NewData("norwich-terrier.jpg", newStubImageReader())
		got := data.Format
		want := "jpeg"
		assertString(t, got, want)
	})

	t.Run("save png format correctly", func(t *testing.T) {
		data, _ := NewData("norwich-terrier.png", newStubImageReader())
		got := data.Format
		want := "png"
		assertString(t, got, want)
	})

	t.Run("return error when invalid fileName is passed", func(t *testing.T) {
		_, got := NewData("norwich-terrier", newStubImageReader())
		want := ErrInvalidFileName
		assertError(t, got, want)
	})

	t.Run("only accept jpg/jpeg and png", func(t *testing.T) {
		_, got := NewData("norwich-terrier.webp", newStubImageReader())
		want := ErrInvalidFormat
		assertError(t, got, want)
	})

	t.Run("return error when decoding image failed", func(t *testing.T) {
		failingReader := &stubImageReader{[]byte("dummy data")}
		_, got := NewData("norwich-terrier.jpg", failingReader)
		want := image.ErrFormat
		assertError(t, got, want)
	})
}
