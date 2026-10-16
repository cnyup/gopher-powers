# Handoff — GopherPowers（great-powers-rts → Go 重写）

> 交接时间：2026-10-16 · 当前 HEAD：`80f67cc`（工作区干净）
> 项目：把 JS/Three.js 的 RTS「大国崛起」重写为 Go 后端 + Vue3 前端
> 源项目 checkout：`/tmp/great-powers-rts`（**临时目录，重启会丢——见「环境重建」**）

---

## 1. 一句话现状

**M1（模拟核心）✅ 完成**、**M2（服务端）✅ 完成**、**M3（前端）进行中（约 70%）**、M4/M5 未开始。

---

## 2. 里程碑账本

| 期 | 内容 | 状态 | 验收证据 |
| --- | --- | --- | --- |
| M1 | internal/game 模拟核心全量移植 | ✅ | 对拍**双剧本全程全等**（经济 92/92、战斗 122/122 快照逐字段一致） |
| M2 | internal/server + protocol + store | ✅ | 双 WS 客户端在 Go 服务端完整对局（battleView 裁剪/指令/结算实测通过） |
| M3 | Vue3 前端 | 🔄 70% | 见下 |
| M4 | go:embed 单二进制 + LAN + Tauri | ⬜ | — |
| M5 | 数据迁移 + 六方压测 | ⬜ | — |

### M3 细分

| 子项 | 状态 | 证据 |
| --- | --- | --- |
| 脚手架 + WS 客户端 + 大厅/房间/战局三视图 | ✅ | `6d18457`，双页真对局 |
| Three.js 战场层 + 选区 + 右键移动 | ✅ | `0f73157`，E2E 实测 |
| 建造放置闭环 + 迷雾可视化 | ✅ | `a8f0383`，E2E：build→ready→放置→落位 |
| **模型平移（45MB GLB + 装配线）** | ✅ | `80f67cc`，探针 meshCount 27→136 |
| 投射物/特效渲染 | ⬜ | 源 `projectile-flight.js` / `realistic-fx.js` 待平移 |
| 小地图 / 框选 / 建筑落位吸附 | ⬜ | — |
| 单机模式（本地打 AI） | ⬜ | 模拟核心已就绪，接线到本地进程即可 |
| 视觉对照验收（与旧版截图比对） | ⬜ | 需要人工/视觉模型 |

---

## 3. 代码地图

```
internal/
  game/          # M1 模拟核心（12.4k 行 Go）——对拍验证全等
    pathfind/    # A* octile（黄金向量测试锁定 JS 库位级语义）
  server/        # M2——view.go(battleView 裁剪) server.go(房间/WS/ticker)
                 #      commands.go(17 指令白名单) ws.go(gorilla 接线)
  protocol/      # roomCode/nickname 校验 + fog 打包
  store/         # modernc/sqlite 账户
cmd/
  gopher-powers/ # 入口：/api/* + /battle-socket + 静态（-root 指向 web/dist）
  parity-replay/ # M1 对拍 harness（Go 侧）
web/             # M3 Vue3+Vite+TS
  src/net/client.ts     # WS 客户端（connect 已 Promise 化处理 cookie 时序）
  src/scene/SceneView.ts # Three.js 战场（模型/迷雾/放置/选区）
  src/scene/models/      # 平移的 13 个视觉 ES 模块
  public/models/         # 54MB：8 GLB + 5 PNG
parity/scripts/  # economy.json / combat.json 对拍剧本
```

---

## 4. 关键运行方式

### 起 Go 服务 + 前端
```bash
cd ~/github/GopherPowers/web && npm run build          # 产出 web/dist
cd .. && go build -o /tmp/gp-server ./cmd/gopher-powers
/tmp/gp-server -port 4190 -root $PWD/web/dist -data /tmp/gp-data
# 浏览器开 http://127.0.0.1:4190
```

### M1 对拍（双剧本验证）
```bash
# JS 侧（需源项目 + 其 node_modules）
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

## 5. 环境重建（重要）

**`/tmp/great-powers-rts` 是源项目 checkout，重启会丢。** 需要时重建：
```bash
git clone https://github.com/chencheng666/great-powers-rts /tmp/great-powers-rts
cd /tmp/great-powers-rts && npm install     # 对拍/mjs 脚本需要 ws、pathfinding、playwright
```
对拍依赖：`node_modules/{ws,pathfinding}`；E2E 依赖 `playwright`（已装 chromium）。

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

---

## 7. 下一步建议（按优先级）

1. **投射物/特效**：平移 `projectile-flight.js` + `realistic-fx.js`，接 SceneView
   （快照 `projectiles`/`effects` 字段已下发）。
2. **单机模式**：Go 入口加 `-solo` 或前端「单机」按钮——本地进程内跑 `game.New`
   打 AI，复用同一模拟核心（方案文档已定：无需 WASM）。
3. **小地图 + 框选**：框选用 raycaster 矩形；小地图用 fog bits 降采样。
4. **M4 go:embed**：`//go:embed web/dist` 收进单二进制；LAN 模式；Tauri 壳。
5. **M5**：guests.sqlite 迁移工具、六方（v3）压测。

---

## 8. 未提交/待办

- 工作区干净，无未提交改动。
- `web/src/scene/models/*.js` 是从源项目平移的 ES 模块（未改写逻辑，仅改资产 URL
  与 data 导入）——后续若做视觉对照发现差异，优先查 URL 与 unitData 表。
- 六方模式（maxPlayers=6）在 server 里显式拒绝（`New` 返回「六方暂未开放」），
  属 M2 已知裁剪范围，未实现。
