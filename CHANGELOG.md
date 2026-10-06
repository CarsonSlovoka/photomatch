# Changelog

本項目所有重大變更均記錄在此文件中

- 此格式基於 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
- 本計畫遵循 [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

