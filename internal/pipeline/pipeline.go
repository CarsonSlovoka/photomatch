package pipeline

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CarsonSlovoka/photomatch/internal/config"
)

const (
	StatusOK       = "成功"
	StatusNoGPS    = "無 GPS"
	StatusTooFar   = "超出距離"
	StatusSkipped  = "跳過"
	StatusReadFail = "讀取失敗"

	dirNoGPS = "未歸類_無GPS"
	dirFar   = "未歸類_超出距離"
)

// GPSFunc 讀一張照片的座標。ok 為 false 表示這張沒有可用 GPS。
type GPSFunc func(path string) (lat, lon float64, ok bool, err error)

type Point struct {
	Name string
	Lat  float64
	Lon  float64
}

type Row struct {
	Original  string
	Status    string
	Matched   string
	Distance  string
	PhotoLat  string
	PhotoLon  string
	OutputRel string
}

type Summary struct {
	Total    int
	OK       int
	NoGPS    int
	TooFar   int
	Skipped  int
	ReadFail int
	Rows     []Row
}

type Options struct {
	Config  config.Config
	ReadGPS GPSFunc
	Now     func() time.Time
}

func Run(opt Options) (Summary, error) {
	if opt.ReadGPS == nil {
		return Summary{}, fmt.Errorf("缺少 GPS 讀取函式")
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	cfg := opt.Config
	if err := cfg.Validate(); err != nil {
		return Summary{}, err
	}

	points, err := readPoints(cfg.Input.Points, cfg.Input.Delimiter, cfg.Input.LatColumn, cfg.Input.LonColumn, cfg.Input.NameColumn)
	if err != nil {
		return Summary{}, err
	}
	if len(points) == 0 {
		return Summary{}, fmt.Errorf("點位 CSV 沒有可用的座標列")
	}

	photos, err := listPhotos(cfg.Input.Photos)
	if err != nil {
		return Summary{}, err
	}

	if err := os.MkdirAll(cfg.Output.Dir, 0o755); err != nil {
		return Summary{}, fmt.Errorf("建立輸出資料夾: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Output.Report), 0o755); err != nil {
		return Summary{}, fmt.Errorf("建立報表資料夾: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Output.Log), 0o755); err != nil {
		return Summary{}, fmt.Errorf("建立 log 資料夾: %w", err)
	}

	logFile, err := os.Create(cfg.Output.Log)
	if err != nil {
		return Summary{}, fmt.Errorf("建立 log: %w", err)
	}
	defer logFile.Close()

	var sum Summary
	nameCount := map[string]int{}
	for _, filename := range photos {
		sum.Total++
		src := filepath.Join(cfg.Input.Photos, filename)
		row := processOne(cfg, points, filename, src, opt.ReadGPS, nameCount, logFile, opt.Now())
		switch row.Status {
		case StatusOK:
			sum.OK++
		case StatusNoGPS:
			sum.NoGPS++
		case StatusTooFar:
			sum.TooFar++
		case StatusSkipped:
			sum.Skipped++
		case StatusReadFail:
			sum.ReadFail++
		}
		sum.Rows = append(sum.Rows, row)
	}
	if sum.Skipped == 0 && sum.ReadFail == 0 {
		fmt.Fprintln(logFile, "沒有因檔名或讀取失敗而跳過的照片")
	}
	if err := writeReport(cfg.Output.Report, sum.Rows); err != nil {
		return sum, err
	}
	return sum, nil
}

func processOne(cfg config.Config, points []Point, filename, src string, readGPS GPSFunc, nameCount map[string]int, logFile *os.File, now time.Time) Row {
	lat, lon, ok, err := readGPS(src)
	if err != nil {
		writeSkip(logFile, now, filename, "", err.Error())
		return Row{Original: filename, Status: StatusReadFail}
	}
	if !ok {
		rel := path.Join(dirNoGPS, filename)
		if err := copyTo(src, filepath.Join(cfg.Output.Dir, rel)); err != nil {
			writeSkip(logFile, now, filename, "", err.Error())
			return Row{Original: filename, Status: StatusReadFail}
		}
		return Row{Original: filename, Status: StatusNoGPS, Matched: "-", OutputRel: rel}
	}

	nearest, dist, found := closest(points, lat, lon)
	distText := ""
	if found {
		distText = fmt.Sprintf("%.1f", dist)
	}
	row := Row{
		Original: filename,
		PhotoLat: fmt.Sprintf("%.6f", lat),
		PhotoLon: fmt.Sprintf("%.6f", lon),
		Distance: distText,
	}
	if !found || dist > cfg.Match.RadiusM {
		rel := path.Join(dirFar, filename)
		if err := copyTo(src, filepath.Join(cfg.Output.Dir, rel)); err != nil {
			writeSkip(logFile, now, filename, "", err.Error())
			return Row{Original: filename, Status: StatusReadFail, PhotoLat: row.PhotoLat, PhotoLon: row.PhotoLon, Distance: distText}
		}
		row.Status = StatusTooFar
		row.Matched = "-"
		row.OutputRel = rel
		return row
	}

	if problem := NameProblem(nearest.Name); problem != "" {
		writeSkip(logFile, now, filename, nearest.Name, problem)
		row.Status = StatusSkipped
		row.Matched = nearest.Name
		return row
	}

	rel, err := destRel(cfg.Match.Rule, nearest.Name, filename, nameCount)
	if err != nil {
		writeSkip(logFile, now, filename, nearest.Name, err.Error())
		row.Status = StatusSkipped
		row.Matched = nearest.Name
		return row
	}
	if err := copyTo(src, filepath.Join(cfg.Output.Dir, filepath.FromSlash(rel))); err != nil {
		writeSkip(logFile, now, filename, nearest.Name, err.Error())
		row.Status = StatusReadFail
		row.Matched = nearest.Name
		return row
	}
	row.Status = StatusOK
	row.Matched = nearest.Name
	row.OutputRel = rel
	return row
}

func destRel(rule, name, filename string, nameCount map[string]int) (string, error) {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	switch rule {
	case config.RulePrefix:
		return name + "_" + filename, nil
	case config.RuleSuffix:
		return base + "_" + name + ext, nil
	case config.RuleReplace:
		nameCount[name]++
		out := name + ext
		if nameCount[name] > 1 {
			out = fmt.Sprintf("%s_%d%s", name, nameCount[name], ext)
		}
		return out, nil
	case config.RuleFolder:
		return path.Join(name, filename), nil
	default:
		return "", fmt.Errorf("不支援的命名規則 %s", rule)
	}
}

func writeSkip(w io.Writer, now time.Time, filename, name, reason string) {
	if name == "" {
		fmt.Fprintf(w, "%s\t%s\t%s\n", now.Format(time.RFC3339), filename, reason)
		return
	}
	fmt.Fprintf(w, "%s\t%s\t井址 %q\t%s\n", now.Format(time.RFC3339), filename, name, reason)
}

func copyTo(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chtimes(dst, info.ModTime(), info.ModTime())
}

func listPhotos(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("讀取照片資料夾: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".jpg" || ext == ".jpeg" {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func readPoints(path, delimiter, latCol, lonCol, nameCol string) ([]Point, error) {
	comma, err := config.ParseDelimiter(delimiter)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("讀取點位 CSV: %w", err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = comma
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析點位 CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("點位 CSV 是空的")
	}
	records[0][0] = strings.TrimPrefix(records[0][0], "\uFEFF")
	idx := map[string]int{}
	for i, h := range records[0] {
		idx[strings.TrimSpace(h)] = i
	}
	latI, ok1 := idx[latCol]
	lonI, ok2 := idx[lonCol]
	nameI, ok3 := idx[nameCol]
	if !ok1 || !ok2 || !ok3 {
		return nil, fmt.Errorf("點位 CSV 找不到欄位 %q、%q 或 %q", latCol, lonCol, nameCol)
	}
	var points []Point
	for n, rec := range records[1:] {
		if latI >= len(rec) || lonI >= len(rec) || nameI >= len(rec) {
			continue
		}
		name := strings.TrimSpace(rec[nameI])
		latText := strings.TrimSpace(rec[latI])
		lonText := strings.TrimSpace(rec[lonI])
		if name == "" && latText == "" && lonText == "" {
			continue
		}
		if name == "" || latText == "" || lonText == "" {
			return nil, fmt.Errorf("點位 CSV 第 %d 列有缺值", n+2)
		}
		lat, err := parseFloat(latText)
		if err != nil {
			return nil, fmt.Errorf("點位 CSV 第 %d 列緯度: %w", n+2, err)
		}
		lon, err := parseFloat(lonText)
		if err != nil {
			return nil, fmt.Errorf("點位 CSV 第 %d 列經度: %w", n+2, err)
		}
		points = append(points, Point{Name: name, Lat: lat, Lon: lon})
	}
	return points, nil
}

func parseFloat(s string) (float64, error) {
	return strconvParse(strings.ReplaceAll(s, ",", ""))
}

func strconvParse(s string) (float64, error) {
	var v float64
	_, err := fmt.Sscan(s, &v)
	if err != nil {
		return 0, fmt.Errorf("無法解析 %q", s)
	}
	return v, nil
}

func writeReport(path string, rows []Row) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("寫入報表: %w", err)
	}
	defer f.Close()
	if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if err := w.Write([]string{"原始檔名", "狀態", "自選欄位內容", "距離(m)", "照片緯度", "照片經度", "輸出相對路徑"}); err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write([]string{row.Original, row.Status, row.Matched, row.Distance, row.PhotoLat, row.PhotoLon, row.OutputRel}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func closest(points []Point, lat, lon float64) (Point, float64, bool) {
	if len(points) == 0 {
		return Point{}, 0, false
	}
	best := points[0]
	bestD := haversine(lat, lon, best.Lat, best.Lon)
	for _, p := range points[1:] {
		d := haversine(lat, lon, p.Lat, p.Lon)
		if d < bestD {
			best = p
			bestD = d
		}
	}
	return best, bestD, true
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earth = 6371000.0
	p1 := lat1 * math.Pi / 180
	p2 := lat2 * math.Pi / 180
	dp := (lat2 - lat1) * math.Pi / 180
	dl := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return earth * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// NameProblem 回傳名稱不能當檔名或資料夾名的原因。空字串表示可用。
func NameProblem(name string) string {
	if strings.TrimSpace(name) == "" {
		return "名稱是空白"
	}
	if strings.TrimRight(name, " .") != name {
		return "名稱結尾含空白或句點"
	}
	if strings.ContainsAny(name, "<>:\"/\\|?*") {
		return "名稱含不能用於檔名的符號"
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return "名稱含控制字元"
		}
	}
	if !utf8.ValidString(name) {
		return "名稱不是有效的文字"
	}
	base := name
	if i := strings.LastIndex(base, "."); i >= 0 {
		base = base[:i]
	}
	switch strings.ToUpper(base) {
	case "CON", "PRN", "AUX", "NUL":
		return "名稱是系統保留字"
	}
	upper := strings.ToUpper(base)
	if len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9' {
		return "名稱是系統保留字"
	}
	return ""
}
