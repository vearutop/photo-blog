package image_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"testing"
)

func TestExifReader_Read(t *testing.T) {
	ex := image.ExifReader{}
	assert.NoError(t, ex.Read("_testdata/IMG_5612.jpg"))
}
