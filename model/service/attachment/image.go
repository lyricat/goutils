package attachment

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/nickalie/go-webpbin"
)

func ConvertStream2Webp(file io.ReadSeeker, ext string) (*bytes.Buffer, error) {
	var img image.Image
	var err error
	if ext == ".jpg" || ext == ".jpeg" {
		img, err = jpeg.Decode(file)
		if err != nil {
			return nil, err
		}
	} else if ext == ".png" {
		img, err = png.Decode(file)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	buf := &bytes.Buffer{}
	if err := webpbin.Encode(buf, img); err != nil {
		return nil, err
	}
	return buf, nil
}
