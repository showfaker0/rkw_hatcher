# 洛克王国世界 · 孵蛋助手

桌面路径：`rkw_hatcher`

| 目录 | 说明 |
|------|------|
| `1_front` | Vue3 + TypeScript 前端 |
| `2_back` | Go + Gin + MySQL 后端 |
| MySQL 库 | `rkw_hatcher_server`（Navicat） |

## 一、初始化数据库

1. 打开 Navicat，选中库 `rkw_hatcher_server`
2. 运行 `2_back/sql/schema.sql`（建表 + 蛋组/性格/奖章种子 + 示例图鉴）

## 二、启动后端

```powershell
cd Desktop\rkw_hatcher\2_back
copy .env.example .env
# 编辑 .env 填入 DB_PASSWORD
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

## 页面

- 配种查询 / 图鉴 / 我的宠物 / 配种线 / 设置（Excel 导入导出）
- 蛋组、性格、奖章字典：**只能在数据库改**，页面只读勾选/下拉

## 字典数据

蛋组 15 个、性格 30 个以游戏内为准，种子见 `2_back/sql/schema.sql`。  
已有库可单独执行：`2_back/sql/seed_egg_groups_natures.sql` 同步/校正字典。

图鉴：`name, evo_chain, best_pvp_natures`（多个性格逗号分隔）, `egg_groups, notes`  
宠物：`species, nickname, gender, nature, is_shiny, notes, medals`（medals 如 `体型:大块头,声音:婉转音`）
