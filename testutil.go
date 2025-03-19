package gato

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"testing"
)

type stubImageReader struct {
	image []byte
}

func (s *stubImageReader) Read(p []byte) (int, error) {
	n := copy(p, s.image)
	if n == len(s.image) {
		return n, io.EOF
	}
	return n, nil
}

func (s *stubImageReader) Close() error {
	return nil
}

func newStubImageData() []byte {
	mockImg := image.NewRGBA(image.Rect(0, 0, 100, 100))
	b := new(bytes.Buffer)
	_ = jpeg.Encode(b, mockImg, nil)
	return b.Bytes()
}

func newStubImageReader() *stubImageReader {
	return &stubImageReader{image: newStubImageData()}
}

func assertString(t testing.TB, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func assertInt(t testing.TB, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func assertError(t testing.TB, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
