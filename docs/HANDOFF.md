# Handoff — GopherPowers（great-powers-rts → Go 重写）

> 交接时间：2026-10-16（第二次交接）· 当前 HEAD：`dd8c24d`（工作区干净）
> 项目：把 JS/Three.js 的 RTS「大国崛起」重写为 Go 后端 + Vue3 前端
> 上一版交接：`80f67cc` 时的 HANDOFF；本版**复验了全部数字**并补了三处原文档没记的事实（见 §8）

---

## 1. 一句话现状

**M1（模拟核心）✅**、**M2（服务端）✅**、**M3（前端）进行中（约 70%）**、M4/M5 未开始。
本次交接实测：`go test ./internal/...` 全绿、`npm run build` 通过（详见 §2 验证列）。

---

## 2. 里程碑账本

| 期 | 内容 | 状态 | 验收证据 |
| --- | --- | --- | --- |
| M1 | internal/game 模拟核心全量移植 | ✅ | 对拍**双剧本全程全等**（经济 92/92、战斗 122/122 快照逐字段一致） |
| M2 | internal/server + protocol + store | ✅ | 双 WS 客户端在 Go 服务端完整对局（battleView 裁剪/指令/结算实测通过） |
| M3 | Vue3 前端 | 🔄 70% | 见 §3 |
| M4 | go:embed 单二进制 + LAN + Tauri | ⬜ | — |
| M5 | 数据迁移 + 六方压测 | ⬜ | — |

**本次（2026-10-16 第二次交接）复验记录**——下个会话可直接重跑这两条：

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/... -count=1` | exit 0；9 个有测试的包全 `ok`（game/data/equip/pathfind/rules/spacing/protocol/server/store），lunar、report 无测试文件 |
| `cd web && npm run build` | exit 0；45 modules，`dist/assets/index-*.js` 769.61 kB（gzip 210.93 kB），615ms |

### M3 细分

| 子项 | 状态 | 证据 |
| --- | --- | --- |
| 脚手架 + WS 客户端 + 大厅/房间/战局三视图 | ✅ | `6d18457`，双页真对局 |
| Three.js 战场层 + 选区 + 右键移动 | ✅ | `0f73157`，E2E 实测 |
| 建造放置闭环 + 迷雾可视化 | ✅ | `a8f0383`，E2E：build→ready→放置→落位 |
| 模型平移（54MB 资产 + 装配线） | ✅ | `80f67cc`，探针 meshCount 27→136 |
| 投射物/特效渲染 | ⬜ | **服务端已下发**（`view.go` 有 `projectiles`/`effects` 且带迷雾裁剪），前端未消费 |
| 小地图 / 框选 / 建筑落位吸附 | ⬜ | `SceneView.ts` 公开面只有 `paintFog/loadAssets/syncView/beginPlace/cancelPlace/flashMarker/resize/render`——无小地图、无矩形框选 |
| 资源点渲染（矿/油/信标） | ⬜ | 前端未消费 `ore/oil/beacons/resourceSites`；且服务端这几项当前是 null，见 §8.2 |
| 单机模式（本地打 AI） | ⬜ | 模拟核心已就绪，接线到本地进程即可 |
| 视觉对照验收（与旧版截图比对） | ⬜ | 需要人工/视觉模型 |

---

## 3. 代码地图（行数为实测）

```
internal/game/     # M1 模拟核心，10250 行 Go——对拍验证全等
  pathfind/        # A* octile（黄金向量测试锁定 JS 库位级语义）
  data/            # 地图/规格表；maps.go 六方图「六域会战」已在，但服务端拒开
internal/server/   # M2，1476 行
  server.go        # 房间/WS/ticker；New() 对 MaxPlayers==6 直接返回「六方暂未开放」
  view.go          # battleView 裁剪：席位重排/敌字段剥离/建筑记忆/fog 打包
  commands.go      # 17 指令白名单
  ws.go            # gorilla 接线
internal/protocol/ # roomCode/nickname 校验 + fog 打包
internal/store/    # modernc/sqlite 账户
cmd/
  gopher-powers/   # 入口：/api/* + /battle-socket + 静态；参数默认见 §5
  parity-replay/   # M1 对拍 harness（Go 侧）
web/src/           # M3 Vue3+Vite+TS，1954 行
  net/client.ts    # WS 客户端（connect 已 Promise 化处理 cookie 时序）
  scene/SceneView.ts # Three.js 战场（模型/迷雾/放置/选区）
  scene/models/    # 平移的 14 个 ES 模块 + unitData.ts + models.d.ts
  views/Battle.vue # HUD：资金/电力/时间/选中/建造/生产/部队列表
web/public/models/ # 54MB（du 实测）：8 GLB + 5 PNG
parity/scripts/    # economy.json / combat.json 对拍剧本
```

---

## 4. 关键运行方式

### 起 Go 服务 + 前端
```bash
cd ~/github/GopherPowers/web && npm run build          # 产出 web/dist（已被 web/.gitignore 忽略）
cd .. && go build -o /tmp/gp-server ./cmd/gopher-powers
/tmp/gp-server -port 4190 -root $PWD/web/dist -data /tmp/gp-data
# 浏览器开 http://127.0.0.1:4190
```

`cmd/gopher-powers/main.go` 的参数默认值（实测）：`-host 127.0.0.1`、`-port 4180`、
`-root dist`、`-data server-data`、`-dev true`（放行 localhost:4173 预览源）。
**注意 `-dev` 默认是 true**——对外发布前要显式关掉。

### M1 对拍（双剧本验证）
```bash
# JS 侧（需源项目 + 其 node_modules；源项目状态见 §6）
cp parity/scripts/*.json /tmp/great-powers-rts/parity/
cd /tmp/great-powers-rts && node parity/replay.mjs parity/economy.json /tmp/js-eco.json
# Go 侧
cd ~/github/GopherPowers && go run ./cmd/parity-replay parity/scripts/economy.json > /tmp/go-eco.json
# 对比：python3 逐快照比对（92/92、122/122 全等为通过）
```

### 测试
```bash
go test ./internal/... -count=1       # 9 包全绿
cd web && npm run build               # vue-tsc + vite（含类型检查）
```

---

## 5. 环境重建（重要——本版有更正）

**`/tmp/great-powers-rts` 当前已是残缺状态，不只是"重启会丢"。** 本次实测：

- `server/` 目录存在但**为空**（`battle-view.mjs` 等已不在）→ 想核对服务端 JS 行为无源可查
- `.git` 目录在但 `git status` 报 `not a git repository` → 无法用 git 恢复
- `src/`、`parity/`、`node_modules/`、`dist/` 仍在（对拍脚本依赖的 `src/game.js` 尚在）

需要完整源时重建：
```bash
rm -rf /tmp/great-powers-rts   # 属破坏性操作，先跟用户确认
git clone https://github.com/chencheng666/great-powers-rts /tmp/great-powers-rts
cd /tmp/great-powers-rts && npm install     # 对拍/mjs 需要 ws、pathfinding；E2E 需要 playwright
```

**已知环境坑**：`web/node_modules` 曾出现 rolldown native binding 损坏
（`Cannot find module '@rolldown/binding-wasm32-wasi'`），修法：
`cd web && rm -rf node_modules package-lock.json && npm install`。

---

## 6. 项目铁律（踩过的坑，务必遵守）

1. **AGENTS.md 的远程工作流不适用本仓**——GopherPowers 不在 mutagen 同步列表里，
   git/测试都在本地执行（已核实）。
2. **浮点语义对齐是 M1 的核心难点**，三条血泪：
   - `math.Sqrt2 - 1` **单表达式会被 Go 编译器常量折叠**到任意精度——必须拆两条语句。
   - 拆了语句还会被**内联 FMA 重融合**（arm64）——必须 `//go:noinline` 的独立乘函数。
   - pathfinding 库的 octile 是 `F*min+max` 形式，代数化简即 1ulp 差、打破堆平局。
   - 黄金向量测试在 `internal/game/pathfind/golden*_test.go`——改动路径逻辑前先跑。
3. **执行时点敏感**：`updateSpecialOrder` 必须在 updateUnit 内（玩家循环之后），
   采集收入早一拍入账会翻转 buildQueue 扣费判定。对齐源 `game.js` 的**调用位置**，
   不只是逻辑。
4. **Vue 响应式**：NetClient 是普通 class，其字段不被 Vue 追踪——必须 provide
   响应式副本（App.vue 已处理）。新增共享状态时沿用此模式。
5. **WS 首连 403**：无 cookie 时升级被拒——`client.session()` 已改为无 user 时
   自动 POST 匿名建号，不要回退。
6. **`web/dist` 不入库**（`web/.gitignore`）——交接给别人时对方必须自己 build，
   否则 `-root` 指过去是空目录。

---

## 7. 下一步建议（按优先级）

1. **投射物/特效渲染**：数据已在快照里（`view.go` 的 `projectiles`/`effects`，含
   `startX/startY` 迷雾裁剪、`targetId/sourceId` 已置空），只需在 `SceneView` 消费。
   源实现是 `projectile-flight.js` + `realistic-fx.js`（注意：源 `server/` 已丢，
   但前端这类文件在 `src/` 侧，仍可对照）。
2. **单机模式**：Go 入口加 `-solo` 或前端「单机」按钮——本地进程内跑 `game.New`
   打 AI，复用同一模拟核心（方案文档已定：无需 WASM）。
3. **资源点渲染 + 补 §8.2 的服务端 site 记忆**：两者要一起做，否则前端拿到全 null。
4. **小地图 + 框选**：框选用 raycaster 矩形；小地图用 fog bits 降采样。
5. **M4 go:embed**：`//go:embed web/dist` 收进单二进制；LAN 模式；Tauri 壳。
6. **M5**：guests.sqlite 迁移工具、六方（v3）压测。

---

## 8. 本次核实出的三处事实（上版文档未记）

### 8.1 服务端 site 记忆是空壳，资源点在快照里恒为 null

`internal/server/view.go:273/275/277`：

```go
func memorySite(memory viewMemory, siteID string) json.RawMessage { return nil }
func setSiteMemory(memory viewMemory, siteID string, v map[string]any) {}
func siteMemory(memory viewMemory, siteID string) map[string]any { return nil }
```

`sites()` 里 `out = append(out, copyAny(siteMemory(memory, id)))` 因此永远 append nil →
`ore`/`oil`/`beacons`/`resourceSites` **数组长度正确但元素全是 `null`**。
建筑记忆（`memory` map）是完整实现的，只有 site 这一路留了 stub。

**影响**：目前前端不消费这四个字段，对局功能不受影响；一旦做资源点渲染立刻踩坑。
**未核实**：源 `server/battle-view.mjs` 已从 /tmp 丢失，无法对照原语义（重新 clone 后可查）。
**建议**：先补 site 记忆再画资源点；补完用"矿被迷雾遮挡 → 应显示记忆位置"的场景验证。

### 8.2 前端完全没有消费资源点与投射物

`web/src` 里 `ore|oil|beacon|resourceSites|projectiles|effects` 零命中（只有 `fog`）。
所以"M2 双 WS 完整对局通过"验的是操作与单位闭环，**视觉上战场只有地面/单位/建筑/迷雾**。

### 8.3 六方是"数据已备、服务端拒开"

`internal/game/data/maps.go:103` 六方图「六域会战」（30 矿/6 油/6 信标，有测试覆盖），
但 `internal/server/server.go:85-86`：

```go
if cfg.MaxPlayers == 6 {
    return nil, fmt.Errorf("六方暂未开放")
}
```

`cmd/gopher-powers/main.go` 硬编码 `MaxPlayers: 2`。M5 要开六方时，改这三处 +
`ValidMap2P`（`internal/game/serverexports.go:11`）。

---

## 9. 未提交/待办

- 工作区干净，无未提交改动（`git status` 实测为空）。
- `web/src/scene/models/*.js` 是从源项目平移的 ES 模块（未改写逻辑，仅改资产 URL
  与 data 导入）——后续若做视觉对照发现差异，优先查 URL 与 unitData 表。
- §8 三项都不是本次改动引入的，是既有实现状态；是否要修、修到哪一层，由下一个会话判断。
