package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/CarsonSlovoka/photomatch/internal/config"
	"github.com/CarsonSlovoka/photomatch/internal/exifgps"
	"github.com/CarsonSlovoka/photomatch/internal/pipeline"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("photomatch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "config.yaml", "設定檔路徑")
	radius := fs.String("radius", "", "覆寫判定半徑（公尺）；省略則用設定檔")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: photomatch [-config 設定檔] [-radius 公尺]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: %v\n", err)
		return 1
	}
	if *radius != "" {
		v, err := strconv.ParseFloat(*radius, 64)
		if err != nil || v < 0 {
			fmt.Fprintf(os.Stderr, "錯誤: -radius 必須是大於等於 0 的數字\n")
			return 2
		}
		cfg.Match.RadiusM = v
	}

	sum, err := pipeline.Run(pipeline.Options{Config: cfg, ReadGPS: exifgps.Read})
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: %v\n", err)
		return 1
	}
	fmt.Printf("完成：共 %d 張，成功 %d，無 GPS %d，超出距離 %d，跳過 %d，讀取失敗 %d\n",
		sum.Total, sum.OK, sum.NoGPS, sum.TooFar, sum.Skipped, sum.ReadFail)
	fmt.Printf("報表：%s\n", cfg.Output.Report)
	fmt.Printf("log：%s\n", cfg.Output.Log)
	return 0
}
