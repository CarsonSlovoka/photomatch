package mapview

import (
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"

	"github.com/CarsonSlovoka/photomatch/internal/pipeline"
)

//go:embed index.html
var pageFS embed.FS

const photoPlaceholder = "__PHOTOS_JSON__"

const (
	thumbMaxEdge = 480
	thumbQuality = 75
)

// Item 是地圖上的一張有座標照片。Label 用輸出相對路徑，不放井址。
type Item struct {
	Original  string  `json:"original"`
	Status    string  `json:"status"`
	OutputRel string  `json:"output"`
	Distance  string  `json:"distance"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Thumb     string  `json:"thumb"`
	Full      string  `json:"full"`
}

// ItemsFromRows 只留有座標、且已複製到輸出資料夾的列。無 GPS 不進地圖。
func ItemsFromRows(rows []pipeline.Row) []Item {
	var items []Item
	for _, row := range rows {
		if row.OutputRel == "" || row.PhotoLat == "" || row.PhotoLon == "" {
			continue
		}
		lat, err1 := strconv.ParseFloat(row.PhotoLat, 64)
		lon, err2 := strconv.ParseFloat(row.PhotoLon, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		items = append(items, Item{
			Original:  row.Original,
			Status:    row.Status,
			OutputRel: row.OutputRel,
			Distance:  row.Distance,
			Lat:       lat,
			Lon:       lon,
			Full:      fullURL(row.OutputRel),
		})
	}
	return items
}

// Write 在輸出資料夾下建立 map/index.html 與縮圖。沒有可標的照片也會寫一頁說明。
func Write(outDir string, rows []pipeline.Row) (string, error) {
	items := ItemsFromRows(rows)
	mapDir := filepath.Join(outDir, "map")
	thumbDir := filepath.Join(mapDir, "thumbs")
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return "", fmt.Errorf("建立地圖資料夾: %w", err)
	}
	for i := range items {
		src := filepath.Join(outDir, filepath.FromSlash(items[i].OutputRel))
		name := fmt.Sprintf("%d.jpg", i+1)
		dst := filepath.Join(thumbDir, name)
		if err := writeThumb(src, dst); err != nil {
			items[i].Thumb = ""
			continue
		}
		items[i].Thumb = "thumbs/" + name
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return "", fmt.Errorf("整理地圖資料: %w", err)
	}
	html, err := page(string(payload))
	if err != nil {
		return "", err
	}
	index := filepath.Join(mapDir, "index.html")
	if err := os.WriteFile(index, []byte(html), 0o644); err != nil {
		return "", fmt.Errorf("寫入地圖頁: %w", err)
	}
	return index, nil
}

// Serve 在本機提供輸出資料夾。只綁 127.0.0.1。
func Serve(outDir, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           http.FileServer(http.Dir(outDir)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return srv.ListenAndServe()
}

func fullURL(rel string) string {
	parts := strings.Split(path.Clean("/"+rel), "/")
	var escaped []string
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		escaped = append(escaped, urlPathEscape(p))
	}
	return "../" + strings.Join(escaped, "/")
}

func urlPathEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

func writeThumb(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	img, err := jpeg.Decode(f)
	if err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err == nil {
		if x, err := exif.Decode(f); err == nil {
			img = applyOrientation(img, orientationOf(x))
		}
	}
	img = fit(img, thumbMaxEdge)
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	return jpeg.Encode(out, img, &jpeg.Options{Quality: thumbQuality})
}

func orientationOf(x *exif.Exif) int {
	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return 1
	}
	v, err := tag.Int(0)
	if err != nil {
		return 1
	}
	return v
}

func applyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return flipH(img)
	case 3:
		return rotate180(img)
	case 4:
		return flipV(img)
	case 5:
		return rotate90(flipH(img))
	case 6:
		return rotate90(img)
	case 7:
		return rotate270(flipH(img))
	case 8:
		return rotate270(img)
	default:
		return img
	}
}

func fit(src image.Image, maxEdge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 || (w <= maxEdge && h <= maxEdge) {
		return src
	}
	nw, nh := w, h
	if w >= h {
		nw = maxEdge
		nh = h * maxEdge / w
	} else {
		nh = maxEdge
		nw = w * maxEdge / h
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

func flipH(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(x, y, src.At(b.Min.X+b.Dx()-1-x, b.Min.Y+y))
		}
	}
	return dst
}

func flipV(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(x, y, src.At(b.Min.X+x, b.Min.Y+b.Dy()-1-y))
		}
	}
	return dst
}

func rotate90(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(b.Dy()-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotate180(src image.Image) image.Image {
	return rotate90(rotate90(src))
}

func rotate270(src image.Image) image.Image {
	return rotate90(rotate180(src))
}

func page(payload string) (string, error) {
	b, err := pageFS.ReadFile("index.html")
	if err != nil {
		return "", fmt.Errorf("讀取地圖頁模板: %w", err)
	}
	tpl := string(b)
	if !strings.Contains(tpl, photoPlaceholder) {
		return "", fmt.Errorf("地圖頁模板缺少 %s", photoPlaceholder)
	}
	return strings.ReplaceAll(tpl, photoPlaceholder, payload), nil
}
