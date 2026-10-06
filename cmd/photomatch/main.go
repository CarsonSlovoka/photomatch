package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/CarsonSlovoka/photomatch/internal/config"
	"github.com/CarsonSlovoka/photomatch/internal/exifgps"
	"github.com/CarsonSlovoka/photomatch/internal/mapview"
	"github.com/CarsonSlovoka/photomatch/internal/pipeline"
)

// version 可由建置時的 -X main.version 覆寫。未覆寫時與目前發行版號相同
var version = "0.0.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("photomatch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "config.yaml", "設定檔路徑")
	radius := fs.String("radius", "", "覆寫判定半徑（公尺）；省略則用設定檔")
	serve := fs.Bool("serve", false, "跑完後在本機開地圖，供瀏覽器核對")
	port := fs.Int("port", 8765, "搭配 -serve 的本機埠")
	showVersion := fs.Bool("version", false, "印出版號後結束")
	fs.BoolVar(showVersion, "V", false, "印出版號後結束")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: photomatch [-config 設定檔] [-radius 公尺] [-serve] [-port 埠] [-V]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Printf("photomatch %s\n", version)
		return 0
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

	index, err := mapview.Write(cfg.Output.Dir, sum.Rows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: %v\n", err)
		return 1
	}
	fmt.Printf("地圖：%s\n", index)
	if !*serve {
		fmt.Println("開啟地圖：加上 -serve，再用瀏覽器打開網址")
		return 0
	}
	if *port < 1 || *port > 65535 {
		fmt.Fprintf(os.Stderr, "錯誤: -port 必須介於 1 到 65535\n")
		return 2
	}
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	fmt.Printf("開啟：http://%s/map/\n", addr)
	fmt.Println("按 Ctrl+C 結束")
	if err := mapview.Serve(cfg.Output.Dir, addr); err != nil {
		fmt.Fprintf(os.Stderr, "錯誤: %v\n", err)
		return 1
	}
	return 0
}
