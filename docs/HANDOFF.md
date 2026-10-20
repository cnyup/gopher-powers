# Handoff — GopherPowers（great-powers-rts → Go 重写）

> 交接时间：2026-10-20（第三次交接）· 当前 HEAD：`d6142fe`（工作区干净）
> 项目：把 JS/Three.js 的 RTS「大国崛起」重写为 Go 后端 + Vue3 前端
> 上一版交接：`dd8c24d` 时的 HANDOFF；本版复验全部数字，M3 推进到约 90%（见 §2/§3），§9 记录本轮修掉的三个真 bug

---

## 1. 一句话现状

**M1（模拟核心）✅**、**M2（服务端）✅**、**M3（前端）约 90%**、**M4/M5 未开始**。
本次交接实测：`go test ./internal/...` 9 包全 pass、`npm run build` exit 0（详见 §2 验证列）。

---

## 2. 里程碑账本

| 期 | 内容 | 状态 | 验收证据 |
| --- | --- | --- | --- |
| M1 | internal/game 模拟核心全量移植 | ✅ | 对拍**双剧本全程全等**（经济 92/92、战斗 122/122 快照逐字段一致） |
| M2 | internal/server + protocol + store | ✅ | 双 WS 客户端在 Go 服务端完整对局（battleView 裁剪/指令/结算实测通过） |
| M3 | Vue3 前端 | 🔄 90% | 见 §3 细分 |
| M4 | go:embed 单二进制 + LAN + Tauri | ⬜ | — |
| M5 | 数据迁移 + 六方压测 | ⬜ | — |

**本次（2026-10-20 第三次交接）复验记录**——下个会话可直接重跑这两条：

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/... -count=1` | exit 0；9 个包全 pass（game/data/equip/pathfind/rules/spacing/protocol/server/store），lunar、report 无测试文件 |
| `npm --prefix web run build` | exit 0；47 modules，`dist/assets/index-*.js` 781.35 kB（gzip 214.96 kB），451ms |
| `npm --prefix web test` | 12/12 pass（`node --test src/scene/*.test.mjs`） |

### M3 细分

| 子项 | 状态 | 证据 |
| --- | --- | --- |
| 脚手架 + WS 客户端 + 大厅/房间/战局三视图 | ✅ | `6d18457`，双页真对局 |
| Three.js 战场层 + 选区 + 右键移动 | ✅ | `0f73157`，E2E 实测 |
| 建造放置闭环 + 迷雾可视化 | ✅ | `a8f0383`，E2E：build→ready→放置→落位 |
| 模型平移（54MB 资产 + 装配线） | ✅ | `80f67cc`，探针 meshCount 27→136 |
| 投射物/特效渲染 | ✅ | `5c60b32`，`FxLayer.ts` 消费 `projectiles/effects`；node --test 6/6（池化/外推/封顶/光束几何/脉冲半径） |
| 资源点渲染（矿/油/信标/资源站） | ✅ | `d21c675`，`siteVisuals.ts` + SceneView `siteMeshes` 差量池；服务端 site 记忆同步补实现（见 §9.1） |
| 单机模式（本地打 AI） | ✅ | `d6142fe`，`create{solo:true}` 自动填 AI 席位；solo_test.go 3/3（AI 席位构造/开局 IsAI/联机房不变） |
| 小地图 / 框选 / 建筑落位吸附 | ⬜ | `SceneView.ts` 仍无小地图、无矩形框选、无吸附 |
| 视觉对照验收（与旧版截图比对） | ⬜ | 需要人工/视觉模型；本轮所有渲染改动**均未做浏览器实拍验证** |

---

## 3. 代码地图（行数为实测）

```
internal/game/     # M1 模拟核心，10250 行 Go——对拍验证全等
  pathfind/        # A* octile（黄金向量测试锁定 JS 库位级语义）
  data/            # 地图/规格表；maps.go 六方图「六域会战」已在，但服务端拒开
  ai.go/ai_place.go # AI 决策：updateAI/queueAIUnit/placeAIBuilding/launchAIWave（单机模式直接复用）
internal/server/   # M2，1759 行
  server.go        # 房间/WS/ticker；New() 对 MaxPlayers==6 直接返回「六方暂未开放」；
                   # create 支持 solo:true；member.userID()/isAI() 处理无 user 的 AI 席位
  view.go          # battleView 裁剪：席位重排/敌字段剥离/建筑+site 记忆/fog 打包
  commands.go      # 17 指令白名单
  ws.go            # gorilla 接线
internal/protocol/ # 203 行，roomCode/nickname 校验 + fog 打包
internal/store/    # 232 行，modernc/sqlite 账户
cmd/               # 215 行
  gopher-powers/   # 入口：/api/* + /battle-socket + 静态；参数默认见 §5
  parity-replay/   # M1 对拍 harness（Go 侧）
web/src/           # M3 Vue3+Vite+TS：13 个 ts/vue 文件 1291 行 + 14 个平移的 models/*.js（合计 2409 行）
  net/client.ts    # WS 客户端（connect 已 Promise 化处理 cookie 时序）；createSolo()
  scene/SceneView.ts # Three.js 战场（模型/迷雾/放置/选区/资源点）
  scene/FxLayer.ts # 投射物池 + 特效（环形脉冲/光束/闪光/收入浮字），帧间插值
  scene/siteVisuals.ts # 资源点纯函数：四类几何/配色/归属着色（可脱离场景测试）
  scene/models/    # 平移的 14 个 ES 模块 + unitData.ts + models.d.ts
  views/Battle.vue # HUD：资金/电力/时间/选中/建造/生产/部队列表
  views/Lobby.vue  # 大厅：建房/加入/「单机对战 AI」一键开局
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

启动后在大厅点「单机对战 AI」即进入单机局（服务端 `create{solo:true}` 自动补 AI 席位，
1 号席位 `controlType=ai`、恒在线恒 ready，玩家 ready 后直接开战，`Multiplayer=false`）。

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
npm --prefix web run build            # vue-tsc + vite（含类型检查）
npm --prefix web test                 # node --test src/scene/*.test.mjs（12 例）
```

---

## 5. 环境重建（本版有更新）

**源项目现在有两个副本，状态不同：**

- `/tmp/great-powers-rts` —— 旧副本，**残缺**：`server/` 目录为空、`.git` 报
  `not a git repository`；`src/`、`parity/`、`node_modules/`、`dist/` 仍在
- `/tmp/gpr-src` —— **本轮新 clone 的完整副本**（`--depth 1`，含 `server/battle-view.mjs`
  等全部服务端源码），补 site 记忆时就是对照它做的

要继续对照源 JS 行为，直接用 `/tmp/gpr-src`；若它也丢了，重建命令：

```bash
git clone --depth 1 https://github.com/chencheng666/great-powers-rts /tmp/gpr-src
cd /tmp/gpr-src && npm install     # 对拍/mjs 需要 ws、pathfinding；E2E 需要 playwright
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

1. **小地图 + 框选 + 建筑落位吸附**：M3 最后一块。框选用 raycaster 矩形求交
   （`pickables` 已有，扩成矩形查询即可）；小地图用 fog bits 降采样画 canvas；
   吸附复用 `place` 指令前的落点校验（`game.ValidPlacement` 一类）。
2. **视觉对照验收**：本轮所有渲染改动（FxLayer/siteVisuals）都**只有逻辑测试，
   没有浏览器实拍**。要起 `-root web/dist` + 前端，真开一局看投射物/特效/资源点。
3. **M4 go:embed**：`//go:embed web/dist` 收进单二进制；LAN 模式（`-host 0.0.0.0`
   + 去掉 `-dev` 默认值）；Tauri 壳。
4. **M5**：guests.sqlite 迁移工具、六方（v3）压测——要开六方需改三处，
   见 §8.3。

---

## 8. 既有事实核实（第二次交接记录，仍然有效）

### 8.1 服务端 site 记忆曾是空壳 → **已于 `d21c675` 修复**

原状：`memorySite`/`setSiteMemory`/`siteMemory` 三个 stub 恒返回 nil，
`sites()` 里 `out = append(out, copyAny(siteMemory(memory, id)))` 永远 append nil →
`ore`/`oil`/`beacons`/`resourceSites` 数组长度正确但元素全 null。

现态：用既有 `siteKey()` 负数键编码（`-h-1`，避开建筑 id 键空间）实现。详见 §9.1。

### 8.2 前端完全没消费资源点与投射物 → **已修复**

`5c60b32` 补投射物/特效（`FxLayer.ts`），`d21c675` 补资源点（`siteVisuals.ts`）。
现在 `web/src` 里这六个字段都有消费方。

### 8.3 六方是"数据已备、服务端拒开"（未变）

`internal/game/data/maps.go:103` 六方图「六域会战」（30 矿/6 油/6 信标，有测试覆盖），
但 `internal/server/server.go:99`：

```go
if cfg.MaxPlayers == 6 {
    return nil, fmt.Errorf("六方暂未开放")
}
```

`cmd/gopher-powers/main.go:28` 硬编码 `MaxPlayers: 2`。M5 要开六方时，改这三处 +
`ValidMap2P`（`internal/game/serverexports.go:12`）。

---

## 9. 本轮（2026-10-20 第三次交接）修掉的三个真 bug

都不是新代码引入的，是照抄源 `battle-view.mjs` 时漏掉的分支，**补 site 记忆后才暴露**。

### 9.1 site 记忆三件套是 stub（`d21c675`）

用既有 `siteKey()` 负数键编码（`-h-1`）实现三个函数。写入前先 `memorySite()!=nil`
判空决定是否刷新记忆，读时 `copyAny(siteMemory(...))`——注意 `copyAny(nil)` 会
返回**非 nil 的空 map**，所以"有元素但字段缺失"是比"元素为 nil"更隐蔽的失败形态。

### 9.2 迷雾剥离 owner 被覆盖回真实归属（`d21c675`）

源 JS 的 `owner(value)` 对 `null` 直通返回 `null`，所以 L38-39 这两步是安全的：

```js
if (!visible && site.owner !== side) { if ('owner' in value) value.owner = null; ... }
if ('owner' in value) value.owner = owner(value.owner);  // owner(null) === null
```

Go 版照抄时写成 `v["owner"] = owner(int(ownerV.(float64)))`——用的是**原始 owner**，
于是刚剥离的 null 被映射回真实归属，敌方占领的矿在被迷雾遮蔽时仍然泄漏归属。
修法：读 `v["owner"]` 当前值再映射，空则跳过。回归测试
`TestBattleViewSitesFogStripsOwner` 锁定该行为。

### 9.3 site 记忆混进 buildings 且会被误删（`d21c675`）

`memory` 现在同时存建筑（int 正数键）和 site（负数键）。两处遍历漏了过滤：

- buildings 输出循环：site 会以"建筑"身份出现在 `buildings` 数组里
- 建筑消失确认循环：`GetBuilding(负数)` 必为 nil，site 坐标一旦可见就被 `delete`

源 JS 用 `typeof id === 'number'` 把 site 挡在外面（它的 site id 是 string）。
Go 版改为 `if id < 0 { continue }`，两处都加。

### 9.4 单机模式的 5 处 nil 解引用（`d6142fe`）

AI 席位没有 `user`，`m.user.ID` 直接解引用会 panic——`TestSoloStartMarksAISide`
实际抓到了开局快照路径的 SIGSEGV。新增 `member.userID()`（nil 安全返回 `""`）
与 `member.isAI()`，快照/member 列表/开战校验/掉线判定/发送处全部改用。

---

## 10. 未提交/待办

- 工作区干净（`git status` 实测为空）。
- **最大遗留：所有前端渲染改动都没有浏览器实拍验证**。FxLayer（投射物/特效）
  与 siteVisuals（资源点）只有 node 逻辑测试 + `vue-tsc` 类型门禁，真实画面
  是否好看、位置是否对，没人看过。
- `web/src/scene/models/*.js` 是从源项目平移的 ES 模块（未改写逻辑，仅改资产 URL
  与 data 导入）——若视觉对照发现差异，优先查 URL 与 unitData 表。
- 单机模式的 AI 阵营是确定性选取（`enemyFactionOf`，russia 优先），没有难度/随机。
- **交付工具限制（踩了两轮）**：`deliver_task` 无法对 Go 的 `_test.go` 逐文件归因
  （连已提交的 `server_test.go` 都报"目标不存在"），node `.mjs` 测试需先在
  `web/package.json` 加 `test` 脚本才可能被识别。绕过方式经用户确认后使用裸
  `git commit`（`d21c675`、`d6142fe` 两次）。

