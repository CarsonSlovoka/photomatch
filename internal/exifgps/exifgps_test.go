package exifgps

import (
	"math"
	"path/filepath"
	"testing"
)

func TestReadGPS(t *testing.T) {
	lat, lon, ok, err := Read(filepath.Join("..", "..", "testdata", "with_gps.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("預期讀到 GPS")
	}
	if math.Abs(lat-25.033090) > 0.00002 || math.Abs(lon-121.565510) > 0.00002 {
		t.Fatalf("座標 = %.6f, %.6f", lat, lon)
	}
}

func TestReadNoGPS(t *testing.T) {
	_, _, ok, err := Read(filepath.Join("..", "..", "testdata", "no_gps.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("沒有 EXIF 時不該回傳座標")
	}
}
