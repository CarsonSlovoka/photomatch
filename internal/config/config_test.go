package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := []byte(`
input:
  points: points.csv
  photos: photos
  lat_column: 緯度N
  lon_column: 經度E
  name_column: 井址
output:
  dir: out
match:
  radius_m: 100
  rule: prefix
`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output.Report != "out/結果.csv" || cfg.Output.Log != "out/skip.log" {
		t.Fatalf("預設輸出路徑不對: %+v", cfg.Output)
	}
	if cfg.Match.RadiusM != 100 {
		t.Fatalf("radius = %v", cfg.Match.RadiusM)
	}
}

func TestRejectBadRule(t *testing.T) {
	cfg := Config{}
	cfg.Input.Points = "a"
	cfg.Input.Photos = "b"
	cfg.Input.LatColumn = "lat"
	cfg.Input.LonColumn = "lon"
	cfg.Input.NameColumn = "name"
	cfg.Output.Dir = "out"
	cfg.Match.RadiusM = 10
	cfg.Match.Rule = "nope"
	if err := cfg.Validate(); err == nil {
		t.Fatal("預期拒絕未知規則")
	}
}
