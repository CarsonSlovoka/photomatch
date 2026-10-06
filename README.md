# photomatch

把資料夾裡的 JPG，依 EXIF GPS 對到點位 CSV，複製到輸出資料夾並依規則改名。沒有視窗。原檔不改；重跑時同名輸出直接覆蓋。檔名無法使用時寫 log 並跳過該張

## 設定

複製 `config.example.yaml` 成 `config.yaml`

```yaml
input:
  points: ./examples/points.csv
  photos: ./photos
  delimiter: ","       # 預設逗號。可改成 "|"、";"、tab，或單一字元
  lat_column: 緯度N
  lon_column: 經度E
  name_column: 井址
output:
  dir: ./out
  report: ./out/結果.csv
  log: ./out/skip.log
match:
  radius_m: 100
  rule: prefix   # prefix | suffix | replace | folder
```

點位檔不是 Excel, 可從 Excel 另存時選 CSV UTF-8。

分隔符預設`逗號`；Tab 分隔可設 `delimiter: tab`，直條可設 `delimiter: "|"`。省略時為逗號。

只掃照片資料夾的第一層，副檔名 `.jpg` / `.jpeg`

命名規則：

- `prefix`：`井址_原檔名`
- `suffix`：`原檔名_井址.jpg`
- `replace`：`井址.jpg`，同一井址的第二張起為 `井址_2.jpg`
- `folder`：不改檔名，放到 `井址/` 資料夾

無 GPS 複製到 `未歸類_無GPS/`。超出半徑複製到 `未歸類_超出距離/`。井址若含 `\ / : * ? " < > |`、控制字元、結尾空白或句點、或是 `CON` / `COM1` 這類保留字，該張不複製，原因寫進 `skip.log`

## 執行

```bash
make build
./bin/photomatch -config config.yaml
./bin/photomatch -config config.yaml -radius 50
./bin/photomatch -V
./bin/photomatch --version
make test
```

Windows 可執行檔：`make build-windows`，產出 `bin/photomatch.exe`

