package exifgps

import (
	"fmt"
	"os"

	"github.com/rwcarlsen/goexif/exif"
)

// Read 讀 JPG 的 EXIF GPS。沒有 EXIF 或沒有座標時 ok 為 false，不算錯誤。
func Read(path string) (lat, lon float64, ok bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false, fmt.Errorf("開啟照片: %w", err)
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return 0, 0, false, nil
	}
	lat, lon, err = x.LatLong()
	if err != nil {
		return 0, 0, false, nil
	}
	return lat, lon, true, nil
}
