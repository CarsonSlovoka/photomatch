package pipeline

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CarsonSlovoka/photomatch/internal/config"
)

func TestHaversineSample(t *testing.T) {
	// 與先前範例相同：W-01 對 IMG_1001 約 14.9 公尺。
	d := haversine(25.033090, 121.565510, 25.033000, 121.565400)
	if d < 14.5 || d > 15.3 {
		t.Fatalf("距離 = %.2f，預期約 14.9", d)
	}
}

func TestNameProblem(t *testing.T) {
	cases := []struct {
		name string
		bad  bool
	}{
		{"W-01", false},
		{"松山樣井", false},
		{"A/B", true},
		{"A:B", true},
		{"A?", true},
		{"  ", true},
		{"W-01 ", true},
		{"CON", true},
		{"COM1", true},
	}
	for _, tc := range cases {
		got := NameProblem(tc.name)
		if tc.bad && got == "" {
			t.Errorf("%q 應該被拒絕", tc.name)
		}
		if !tc.bad && got != "" {
			t.Errorf("%q 不該被拒絕: %s", tc.name, got)
		}
	}
}

func TestRunMatchRenameSkipAndOverwrite(t *testing.T) {
	root := t.TempDir()
	points := filepath.Join(root, "points.csv")
	photos := filepath.Join(root, "photos")
	out := filepath.Join(root, "out")
	if err := os.Mkdir(photos, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, points, "項次,井址,緯度N,經度E\n1,W-01,25.033000,121.565400\n2,A/B,25.034800,121.568200\n3,W-03,25.040200,121.572000\n")
	writeFile(t, filepath.Join(photos, "near.jpg"), "near-v1")
	writeFile(t, filepath.Join(photos, "also.jpg"), "also")
	writeFile(t, filepath.Join(photos, "badname.jpg"), "bad")
	writeFile(t, filepath.Join(photos, "far.jpg"), "far")
	writeFile(t, filepath.Join(photos, "nogps.jpg"), "nogps")
	writeFile(t, filepath.Join(photos, "note.txt"), "ignore")

	gps := map[string][2]float64{
		"near.jpg":    {25.033090, 121.565510},
		"also.jpg":    {25.033210, 121.565620},
		"badname.jpg": {25.034920, 121.568310},
		"far.jpg":     {25.047000, 121.580000},
	}
	cfg := testConfig(points, photos, out, config.RulePrefix, 100)
	sum, err := Run(Options{Config: cfg, ReadGPS: fakeGPS(photos, gps), Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if sum.OK != 2 || sum.Skipped != 1 || sum.TooFar != 1 || sum.NoGPS != 1 || sum.Total != 5 {
		t.Fatalf("摘要不對: %+v", sum)
	}

	assertFile(t, filepath.Join(out, "W-01_near.jpg"), "near-v1")
	assertFile(t, filepath.Join(out, "W-01_also.jpg"), "also")
	if _, err := os.Stat(filepath.Join(out, "A/B_badname.jpg")); !os.IsNotExist(err) {
		t.Fatal("不合法檔名不該被寫出")
	}
	assertFile(t, filepath.Join(out, "未歸類_超出距離", "far.jpg"), "far")
	assertFile(t, filepath.Join(out, "未歸類_無GPS", "nogps.jpg"), "nogps")
	if _, err := os.Stat(filepath.Join(out, "note.txt")); !os.IsNotExist(err) {
		t.Fatal("非 jpg 不該被複製")
	}

	logText := readText(t, cfg.Output.Log)
	if !strings.Contains(logText, "badname.jpg") || !strings.Contains(logText, "A/B") {
		t.Fatalf("log 沒有跳過紀錄: %s", logText)
	}
	report := readReport(t, cfg.Output.Report)
	if report[0][0] != "原始檔名" {
		t.Fatalf("報表標題不對: %v", report[0])
	}
	status := map[string]string{}
	for _, row := range report[1:] {
		status[row[0]] = row[1]
	}
	if status["near.jpg"] != StatusOK || status["badname.jpg"] != StatusSkipped || status["far.jpg"] != StatusTooFar || status["nogps.jpg"] != StatusNoGPS {
		t.Fatalf("狀態不對: %v", status)
	}

	// 重跑覆蓋舊檔，不另存。
	writeFile(t, filepath.Join(photos, "near.jpg"), "near-v2")
	if _, err := Run(Options{Config: cfg, ReadGPS: fakeGPS(photos, gps), Now: fixedNow}); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(out, "W-01_near.jpg"), "near-v2")
}

func TestReplaceAndFolderRules(t *testing.T) {
	root := t.TempDir()
	points := filepath.Join(root, "points.csv")
	photos := filepath.Join(root, "photos")
	out := filepath.Join(root, "out")
	os.Mkdir(photos, 0o755)
	writeFile(t, points, "井址,緯度N,經度E\nW-01,25.033000,121.565400\n")
	writeFile(t, filepath.Join(photos, "a.jpg"), "a")
	writeFile(t, filepath.Join(photos, "b.jpg"), "b")
	gps := map[string][2]float64{
		"a.jpg": {25.033090, 121.565510},
		"b.jpg": {25.033100, 121.565500},
	}

	cfg := testConfig(points, photos, out, config.RuleReplace, 100)
	sum, err := Run(Options{Config: cfg, ReadGPS: fakeGPS(photos, gps)})
	if err != nil {
		t.Fatal(err)
	}
	if sum.OK != 2 {
		t.Fatalf("replace 成功數 = %d", sum.OK)
	}
	assertFile(t, filepath.Join(out, "W-01.jpg"), "a")
	assertFile(t, filepath.Join(out, "W-01_2.jpg"), "b")

	out2 := filepath.Join(root, "out-folder")
	cfg = testConfig(points, photos, out2, config.RuleFolder, 100)
	if _, err := Run(Options{Config: cfg, ReadGPS: fakeGPS(photos, gps)}); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(out2, "W-01", "a.jpg"), "a")
	assertFile(t, filepath.Join(out2, "W-01", "b.jpg"), "b")
}

func TestSuffixRule(t *testing.T) {
	root := t.TempDir()
	points := filepath.Join(root, "points.csv")
	photos := filepath.Join(root, "photos")
	out := filepath.Join(root, "out")
	os.Mkdir(photos, 0o755)
	writeFile(t, points, "井址,緯度N,經度E\nW-01,25.0,121.0\n")
	writeFile(t, filepath.Join(photos, "shot.jpg"), "x")
	cfg := testConfig(points, photos, out, config.RuleSuffix, 100)
	gps := map[string][2]float64{"shot.jpg": {25.0, 121.0}}
	if _, err := Run(Options{Config: cfg, ReadGPS: fakeGPS(photos, gps)}); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(out, "shot_W-01.jpg"), "x")
}

func TestOutsideRadiusNotMatched(t *testing.T) {
	root := t.TempDir()
	points := filepath.Join(root, "points.csv")
	photos := filepath.Join(root, "photos")
	out := filepath.Join(root, "out")
	os.Mkdir(photos, 0o755)
	writeFile(t, points, "井址,緯度N,經度E\nW-01,25.033000,121.565400\n")
	writeFile(t, filepath.Join(photos, "far.jpg"), "far")
	cfg := testConfig(points, photos, out, config.RulePrefix, 10)
	gps := map[string][2]float64{"far.jpg": {25.033090, 121.565510}}
	sum, err := Run(Options{Config: cfg, ReadGPS: fakeGPS(photos, gps)})
	if err != nil {
		t.Fatal(err)
	}
	if sum.TooFar != 1 || sum.OK != 0 {
		t.Fatalf("10 公尺半徑不該配對: %+v", sum)
	}
}

func testConfig(points, photos, out, rule string, radius float64) config.Config {
	var cfg config.Config
	cfg.Input.Points = points
	cfg.Input.Photos = photos
	cfg.Input.LatColumn = "緯度N"
	cfg.Input.LonColumn = "經度E"
	cfg.Input.NameColumn = "井址"
	cfg.Output.Dir = out
	cfg.Output.Report = filepath.Join(out, "結果.csv")
	cfg.Output.Log = filepath.Join(out, "skip.log")
	cfg.Match.RadiusM = radius
	cfg.Match.Rule = rule
	return cfg
}

func fakeGPS(dir string, known map[string][2]float64) GPSFunc {
	return func(p string) (float64, float64, bool, error) {
		name := filepath.Base(p)
		ll, ok := known[name]
		if !ok {
			return 0, 0, false, nil
		}
		return ll[0], ll[1], true, nil
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 10, 5, 18, 0, 0, 0, time.UTC)
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("讀不到 %s: %v", path, err)
	}
	if string(b) != want {
		t.Fatalf("%s 內容 = %q，預期 %q", path, b, want)
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func readReport(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	bom := make([]byte, 3)
	if _, err := f.Read(bom); err != nil {
		t.Fatal(err)
	}
	if string(bom) != "\uFEFF" {
		t.Fatal("報表缺少 UTF-8 BOM")
	}
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
