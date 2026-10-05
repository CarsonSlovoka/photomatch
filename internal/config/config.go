package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	RulePrefix  = "prefix"
	RuleSuffix  = "suffix"
	RuleReplace = "replace"
	RuleFolder  = "folder"
)

type Config struct {
	Input struct {
		Points     string `yaml:"points"`
		Photos     string `yaml:"photos"`
		LatColumn  string `yaml:"lat_column"`
		LonColumn  string `yaml:"lon_column"`
		NameColumn string `yaml:"name_column"`
	} `yaml:"input"`
	Output struct {
		Dir    string `yaml:"dir"`
		Report string `yaml:"report"`
		Log    string `yaml:"log"`
	} `yaml:"output"`
	Match struct {
		RadiusM float64 `yaml:"radius_m"`
		Rule    string  `yaml:"rule"`
	} `yaml:"match"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("讀取設定檔: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("解析設定檔: %w", err)
	}
	cfg.normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) normalize() {
	c.Input.LatColumn = strings.TrimSpace(c.Input.LatColumn)
	c.Input.LonColumn = strings.TrimSpace(c.Input.LonColumn)
	c.Input.NameColumn = strings.TrimSpace(c.Input.NameColumn)
	c.Match.Rule = strings.ToLower(strings.TrimSpace(c.Match.Rule))
	if c.Output.Report == "" && c.Output.Dir != "" {
		c.Output.Report = c.Output.Dir + "/結果.csv"
	}
	if c.Output.Log == "" && c.Output.Dir != "" {
		c.Output.Log = c.Output.Dir + "/skip.log"
	}
}

func (c Config) Validate() error {
	if c.Input.Points == "" || c.Input.Photos == "" {
		return fmt.Errorf("設定檔缺少 input.points 或 input.photos")
	}
	if c.Input.LatColumn == "" || c.Input.LonColumn == "" || c.Input.NameColumn == "" {
		return fmt.Errorf("設定檔缺少緯度、經度或命名欄位")
	}
	if c.Output.Dir == "" {
		return fmt.Errorf("設定檔缺少 output.dir")
	}
	if c.Match.RadiusM < 0 {
		return fmt.Errorf("判定半徑不可為負數")
	}
	switch c.Match.Rule {
	case RulePrefix, RuleSuffix, RuleReplace, RuleFolder:
		return nil
	default:
		return fmt.Errorf("不支援的命名規則 %q，可用 prefix、suffix、replace、folder", c.Match.Rule)
	}
}
