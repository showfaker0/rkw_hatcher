# 洛克王国世界 · 孵蛋助手

桌面路径：`rkw_hatcher`

| 目录 | 说明 |
|------|------|
| `1_front` | Vue3 + TypeScript 前端 |
| `2_back` | Go + Gin + SQLite 后端 |
| `release` | 一键打包产物（`rkw_hatcher.exe` + `rkw_hatcher.db`） |

数据存在本地 SQLite 文件，**不需要 MySQL**。

## 一、数据库

首次启动会自动建库、建表，并写入蛋组 / 性格 / 奖章种子。

- 开发：`2_back/data/rkw_hatcher.db`
- 打包：与 `rkw_hatcher.exe` 同目录的 `rkw_hatcher.db`

可选：在 `.env` 里设置 `DB_PATH` 覆盖路径。

## 二、启动后端

```powershell
cd Desktop\rkw_hatcher\2_back
copy .env.example .env
$env:Path = "C:\Program Files\Go\bin;" + $env:Path
$env:GOPROXY = "https://goproxy.cn,direct"
go run ./cmd/server
```

默认监听 `:3070`。

## 三、启动前端

```powershell
cd Desktop\rkw_hatcher\1_front
npm install
npm run dev -- --host
```

浏览器打开终端提示的地址。手机需同一 WiFi，访问电脑局域网 IP。

## 四、一键打包

```powershell
cd Desktop\rkw_hatcher
.\build-exe.ps1
```

双击 `release\rkw_hatcher.exe` 即可使用。

## 页面

- 配种查询 / 图鉴 / 我的宠物 / 配种线 / 设置（Excel 导入导出）
- 蛋组、性格、奖章字典：库内维护（首次自动种子），页面只读勾选/下拉

## 字典与业务数据

图鉴：`name, evo_chain, best_pvp_natures`（多个性格逗号分隔）, `egg_groups, notes`  
宠物：`species, nickname, gender, nature, is_shiny, notes, medals`（medals 如 `体型:大块头,声音:婉转音`）
