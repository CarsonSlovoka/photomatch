# Changelog

本項目所有重大變更均記錄在此文件中

- 此格式基於 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
- 本計畫遵循 [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-10-06

目前發行版

### Added

- `-V` 與 `--version`：印出版號後結束。未覆寫時為 `0.1.0`；建置可用 `-X main.version` 注入，發行標籤會去掉開頭的 `v`
- 點位分隔符 `input.delimiter`。預設逗號。可設單一字元，或 `comma`、`tab`／`tsv`、`pipe`、`semicolon`；字面 `\t` 也視為定位字元。換行不能當分隔符
- 每次執行後在輸出資料夾寫地圖核對頁 `map/index.html` 與縮圖 `map/thumbs/`。沒有可標的照片也會寫一頁說明
- `-serve`：跑完後只綁 `127.0.0.1` 提供輸出資料夾。預設埠 `8765`，可用 `-port` 改（1–65535）。瀏覽器開印出的 `http://127.0.0.1:8765/map/`。不加 `-serve` 仍會寫地圖檔
- 地圖只標有 GPS、且已複製出去的照片（成功、超出距離）。清單與彈出視窗用輸出相對路徑，不顯示井址；含縮圖、狀態、距離、原始檔名、座標，可開輸出檔
- 縮圖依 EXIF Orientation 轉正，長邊最多 480、JPEG 品質 75。縮圖失敗仍保留該點
- 圖磚為 OpenStreetMap，經 MapLibre GL 顯示，頁面保留出處。給本機核對，不要把位址拿去對外服務
- 頁面範本 `internal/mapview/index.html`，以 `go:embed` 嵌入；照片 JSON 取代 `__PHOTOS_JSON__`

### Changed

- 點位檔改為分隔文字，不再限逗號 CSV。從 Excel 另存時仍用 CSV UTF-8。報表維持逗號 CSV、UTF-8 含 BOM
- 結束時一併印出地圖路徑；未加 `-serve` 時提示如何開本機頁面

## [0.0.0] - 2026-10-05

### Added

- 命令列工具 `photomatch`：依照片 EXIF GPS 對到點位 CSV 的最近點，複製到輸出資料夾並改名。沒有視窗，不修改原檔
- YAML 設定檔。由 `config.example.yaml` 複製成 `config.yaml`，指定點位檔、照片資料夾、緯度／經度／命名欄位、輸出資料夾、報表、log、判定半徑與命名規則
- 參數 `-config`（預設 `config.yaml`）與 `-radius`（覆寫判定半徑，須大於等於 0）
- 點位讀取：CSV（含 UTF-8 BOM），欄位名稱可設定，預設為 `緯度N`、`經度E`、`井址`。空列略過；缺名稱或座標則中止
- 照片掃描：只讀照片資料夾第一層的 `.jpg`／`.jpeg`，不進入子資料夾。以 `goexif` 讀 GPS；沒有 EXIF 或沒有座標不算錯誤
- 比對：以 Haversine 公式（地球半徑 6,371,000 公尺）找最近點。距離在 `radius_m` 以內（預設 100）視為成功
- 命名規則 `prefix`（`井址_原檔名`）、`suffix`（`原檔名_井址` 加原副檔名）、`replace`（`井址` 加原副檔名，同一井址第二張起為 `井址_2`）、`folder`（放到 `井址/`，不改檔名）
- 無 GPS 複製到 `未歸類_無GPS/`；有座標但超出半徑複製到 `未歸類_超出距離/`
- 井址不能當檔名或資料夾名時不複製，原因寫入 `skip.log`。包含禁用符號、控制字元、結尾空白或句點，以及 `CON`、`PRN`、`AUX`、`NUL`、`COM1`–`COM9`、`LPT1`–`LPT9`
- 同名輸出直接覆蓋，並保留原檔修改時間。每次執行重建 `skip.log`；沒有跳過或讀取失敗時寫入一行說明
- 報表為 UTF-8（含 BOM）的 CSV，預設 `./out/結果.csv`。欄位為原始檔名、狀態、自選欄位內容、距離(m)、照片緯度、照片經度、輸出相對路徑。狀態為成功、無 GPS、超出距離、跳過、讀取失敗
- 結束時在標準輸出印出張數統計，以及報表與 log 路徑
- `Makefile`：`build`、`test`、`vet`、`ci`、`run`、`build-windows`、`build-linux`
- GitHub Actions：推送與 pull request 執行 vet、測試與建置；推送 `v*` 標籤時發布 linux/amd64、windows/amd64、darwin/amd64、darwin/arm64
- 範例點位 `examples/points.csv`，以及含 GPS、不含 GPS 的測試照片

