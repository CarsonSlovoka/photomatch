package mapview

import (
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
	index := filepath.Join(mapDir, "index.html")
	if err := os.WriteFile(index, []byte(page(string(payload))), 0o644); err != nil {
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

func page(payload string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-Hant">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>photomatch 地圖核對</title>
<link href="https://unpkg.com/maplibre-gl@4.7.1/dist/maplibre-gl.css" rel="stylesheet">
<style>
  html, body { margin: 0; height: 100%%; font-family: "Segoe UI", "Noto Sans TC", sans-serif; color: #1c1917; }
  body { display: flex; min-height: 100%%; }
  #side { width: 320px; flex: 0 0 320px; display: flex; flex-direction: column; border-right: 1px solid #e7e5e4; background: #fafaf9; }
  #side header { padding: 14px 16px 10px; border-bottom: 1px solid #e7e5e4; }
  #side h1 { margin: 0; font-size: 16px; font-weight: 650; }
  #side p { margin: 6px 0 0; font-size: 12px; color: #57534e; line-height: 1.45; }
  #list { overflow: auto; flex: 1; }
  .item { display: block; width: 100%%; text-align: left; border: 0; border-bottom: 1px solid #e7e5e4; background: transparent; padding: 10px 16px; cursor: pointer; }
  .item:hover, .item.active { background: #f5f5f4; }
  .item .path { font-size: 13px; word-break: break-all; }
  .item .meta { margin-top: 3px; font-size: 12px; color: #78716c; }
  .dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%%; margin-right: 6px; }
  .ok { background: #1f7a4d; }
  .far { background: #d9822b; }
  #map { flex: 1; min-width: 0; }
  .maplibregl-popup-content { padding: 10px 12px; border-radius: 8px; }
  .popup img { display: block; width: 280px; max-width: 70vw; height: auto; border-radius: 4px; background: #e7e5e4; }
  .popup .path { margin: 8px 0 0; font-size: 13px; font-weight: 650; word-break: break-all; }
  .popup .meta { margin: 4px 0 0; font-size: 12px; color: #57534e; }
  .popup a { color: #1d4ed8; }
  #empty { padding: 16px; color: #57534e; font-size: 14px; }
  @media (max-width: 800px) {
    body { flex-direction: column; }
    #side { width: auto; flex-basis: 38vh; border-right: 0; border-bottom: 1px solid #e7e5e4; }
    #map { min-height: 62vh; }
  }
</style>
</head>
<body>
<aside id="side">
  <header>
    <h1>照片位置</h1>
    <p>只顯示有 GPS 的照片。名稱用輸出路徑，不顯示井址。點清單或地圖上的點可看縮圖。</p>
  </header>
  <div id="list"></div>
</aside>
<div id="map"></div>
<script src="https://unpkg.com/maplibre-gl@4.7.1/dist/maplibre-gl.js"></script>
<script>
const photos = %s;
const list = document.getElementById('list');
const map = new maplibregl.Map({
  container: 'map',
  style: {
    version: 8,
    sources: {
      osm: {
        type: 'raster',
        tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
        tileSize: 256,
        attribution: '&copy; OpenStreetMap contributors'
      }
    },
    layers: [{ id: 'osm', type: 'raster', source: 'osm' }]
  },
  center: [121.5, 25.0],
  zoom: 6
});
map.addControl(new maplibregl.NavigationControl(), 'top-right');

function popupHTML(p) {
  const img = p.thumb ? '<img src="' + p.thumb + '" alt="">' : '<div class="meta">縮圖失敗，仍可開原圖</div>';
  const dist = p.distance ? p.distance + ' m' : '—';
  return '<div class="popup">' + img +
    '<div class="path">' + escapeHTML(p.output) + '</div>' +
    '<div class="meta">' + escapeHTML(p.status) + ' · 距離 ' + escapeHTML(dist) + '<br>原始檔名 ' + escapeHTML(p.original) +
    '<br>' + p.lat.toFixed(6) + ', ' + p.lon.toFixed(6) +
    '<br><a href="' + p.full + '" target="_blank" rel="noopener">開啟輸出檔</a></div></div>';
}
function escapeHTML(s) {
  return String(s).replace(/[&<>"']/g, function (c) {
    return ({ '&': '&', '<': '<', '>': '>', '"': '"', "'": '&#39;' })[c];
  });
}
const markers = photos.map(function (p, i) {
  const el = document.createElement('button');
  el.type = 'button';
  el.title = p.output;
  el.style.cssText = 'width:16px;height:16px;border-radius:50%%;border:2px solid #fff;box-shadow:0 0 0 1px rgba(0,0,0,.35);padding:0;cursor:pointer;background:' + (p.status === '超出距離' ? '#d9822b' : '#1f7a4d');
  const marker = new maplibregl.Marker({ element: el }).setLngLat([p.lon, p.lat]).addTo(map);
  const popup = new maplibregl.Popup({ offset: 16, maxWidth: '320px' }).setHTML(popupHTML(p));
  marker.setPopup(popup);
  el.addEventListener('click', function () { activate(i); });
  return { marker: marker, popup: popup };
});

photos.forEach(function (p, i) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'item';
  btn.innerHTML = '<div class="path"><span class="dot ' + (p.status === '超出距離' ? 'far' : 'ok') + '"></span>' + escapeHTML(p.output) + '</div>' +
    '<div class="meta">' + escapeHTML(p.status) + (p.distance ? ' · ' + escapeHTML(p.distance) + ' m' : '') + '</div>';
  btn.addEventListener('click', function () { activate(i, true); });
  list.appendChild(btn);
});
if (!photos.length) {
  list.innerHTML = '<div id="empty">這次沒有可標在地圖上的照片。</div>';
}

function activate(i, fly) {
  document.querySelectorAll('.item').forEach(function (el, n) { el.classList.toggle('active', n === i); });
  const p = photos[i];
  if (fly) map.flyTo({ center: [p.lon, p.lat], zoom: Math.max(map.getZoom(), 16) });
  markers[i].popup.setLngLat([p.lon, p.lat]).addTo(map);
  const btn = list.children[i];
  if (btn && btn.scrollIntoView) btn.scrollIntoView({ block: 'nearest' });
}
map.on('load', function () {
  if (!photos.length) return;
  const bounds = new maplibregl.LngLatBounds();
  photos.forEach(function (p) { bounds.extend([p.lon, p.lat]); });
  map.fitBounds(bounds, { padding: 48, maxZoom: 16 });
});
</script>
</body>
</html>
`, payload)
}
