package mapview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CarsonSlovoka/photomatch/internal/pipeline"
)

func TestItemsFromRowsSkipsMissingGPS(t *testing.T) {
	items := ItemsFromRows([]pipeline.Row{
		{Original: "a.jpg", Status: pipeline.StatusOK, OutputRel: "井A_a.jpg", PhotoLat: "25.033090", PhotoLon: "121.565510", Distance: "14.9"},
		{Original: "b.jpg", Status: pipeline.StatusNoGPS, OutputRel: "未歸類_無GPS/b.jpg"},
		{Original: "c.jpg", Status: pipeline.StatusTooFar, OutputRel: "未歸類_超出距離/c.jpg", PhotoLat: "24.100000", PhotoLon: "120.500000", Distance: "250.0"},
		{Original: "d.jpg", Status: pipeline.StatusSkipped, PhotoLat: "25.000000", PhotoLon: "121.000000", Matched: "不該出現"},
	})
	if len(items) != 2 {
		t.Fatalf("要 2 筆有座標的照片，得到 %d", len(items))
	}
	if items[0].OutputRel != "井A_a.jpg" || items[1].Status != pipeline.StatusTooFar {
		t.Fatalf("內容不對: %+v", items)
	}
	blob, _ := os.ReadFile(filepath.Join("mapview.go"))
	if strings.Contains(string(blob), "不該出現") {
		t.Fatal("測試資料不應寫進程式")
	}
}

func TestWritePageAndThumb(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "with_gps.jpg")
	rel := "井A_with_gps.jpg"
	if err := copyFile(src, filepath.Join(dir, rel)); err != nil {
		t.Fatal(err)
	}
	index, err := Write(dir, []pipeline.Row{
		{Original: "with_gps.jpg", Status: pipeline.StatusOK, OutputRel: rel, PhotoLat: "25.033090", PhotoLon: "121.565510", Distance: "14.9", Matched: "不顯示的井址"},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	if !strings.Contains(page, "井A_with_gps.jpg") {
		t.Fatal("地圖頁沒有輸出路徑")
	}
	if strings.Contains(page, "不顯示的井址") {
		t.Fatal("地圖頁不該出現井址")
	}
	if !strings.Contains(page, "tile.openstreetmap.org") {
		t.Fatal("地圖頁沒有 OpenStreetMap 圖磚")
	}
	if _, err := os.Stat(filepath.Join(dir, "map", "thumbs", "1.jpg")); err != nil {
		t.Fatal(err)
	}
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
