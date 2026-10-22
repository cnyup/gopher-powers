# Handoff — GopherPowers（great-powers-rts → Go 重写）

> 交接时间：2026-10-21（第五次交接）· 当前 HEAD：`436f807`（工作区干净）
> 项目：把 JS/Three.js 的 RTS「大国崛起」重写为 Go 后端 + Vue3 前端
> 上一版交接：`4d687c2` 时的 HANDOFF。本版**纠正上一版一处错误结论**：
> M3 不是「功能面完成」——**终局闭环整块缺失**（见 §12），已用浏览器实测证伪。

---

## 1. 一句话现状

**M1（模拟核心）✅**、**M2（服务端）✅**、**M3（前端）渲染面完成、终局闭环缺失**、**M4/M5 未开始**。
本次交接实测：`go test ./internal/...` 11 包全 ok、`npm test` 36/36、`npm run build` exit 0
（详见 §2 验证列）。

**M3 真正的缺口只有两个，且都不是"没实现"，是"没接线"**：

1. **终局无任何呈现**（§12）——打完一局/认输，前端无胜负界面，直接弹回大厅。
   服务端 `winner` 已下发、战报包 `report.go` 已移植，**两边都没被前端消费**。
2. **与旧版 JS 前端的视觉像素对照**——渲染已实拍验证过存在性与着色，但未做像素级比对。

---

## 2. 里程碑账本

| 期 | 内容 | 状态 | 验收证据 |
| --- | --- | --- | --- |
| M1 | internal/game 模拟核心全量移植 | ✅ | 对拍**双剧本全程全等**（经济 92/92、战斗 122/122 快照逐字段一致） |
| M2 | internal/server + protocol + store | ✅ | 双 WS 客户端在 Go 服务端完整对局（battleView 裁剪/指令/结算实测通过） |
| M3 | Vue3 前端 | 🔄 渲染面完成，终局缺失 | 见 §2 细分与 §12 |
| M4 | go:embed 单二进制 + LAN + Tauri | ⬜ | — |
| M5 | 数据迁移 + 六方压测 | ⬜ | — |

**本次（2026-10-21 第五次交接）复验记录**——下个会话可直接重跑这三条：

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/... -count=1` | exit 0；11 个包全 ok（lunar、report 无测试文件） |
| `npm --prefix web test` | 36/36 pass（FxLayer 6 + siteVisuals 7 + selection 7 + placement 16） |
| `npm --prefix web run build` | exit 0；49 modules，`dist/assets/index-*.js` 787.37 kB（gzip 216.88 kB），351ms |

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
| 小地图 / 框选 | ✅ | `7d9de93`，`selection.ts` 纯函数 + `SceneView` 拖拽/小地图 canvas |
| 建筑落位红绿着色 | ✅ | `4d687c2`，`placement.ts` 复刻 canPlace（16/16）+ 服务端下发 `placeRules` |
| **终局结算界面** | ⬜ | **整块缺失**——`winner` 已下发但前端零消费，战报包 `report.go` 未被 `battleView` 调用（§12） |
| 视觉对照验收（与旧版截图比对） | ⬜ | 渲染已实拍验证过存在性/着色（§11），未与旧版像素级对照，需人工或视觉模型 |

> 注：上一版把「建筑落位吸附」列为待办——**实测源项目根本没有吸附**
> （`render.js:1183` 直接把指针世界坐标传给 `canPlace`/`placeBuilding`）。
> 源项目有而我们缺的是**合法性红绿着色**与**失败反馈**，本轮补的是这个，不是吸附。

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
go test ./internal/... -count=1       # 11 包全绿
npm --prefix web run build            # vue-tsc + vite（含类型检查）
npm --prefix web test                 # node --test src/scene/*.test.mjs（36 例）
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

1. **终局闭环（M3 最后一块，也是唯一的功能缺口）**：见 §12 的三步方案。
   服务端补下发 `battleReport`（照 `battle-view.mjs:47`），前端补结算层读
   `view.winner`。做完 M3 才算真正"打完整局"。
2. **视觉对照验收**：渲染已实拍（§11），但未与旧版 JS 前端做像素级比对。
   需把源项目也跑起来，同图同机位截图对照，或人工目视。
3. **M4 go:embed**：`//go:embed web/dist` 收进单二进制；LAN 模式（`-host 0.0.0.0`
   + 去掉 `-dev` 默认值）；Tauri 壳。
   **注意 `-dev` 默认是 true**（`main.go:27`），对外发布前必须显式关掉。
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
- **渲染实拍已于第四次交接完成**（§11）：投射物/特效/资源点/小地图/落位着色的
  存在性与颜色都已截图验证。仍未做的是**与旧版 JS 前端的像素级对照**。
- **终局闭环缺失**（§12）：M3 唯一的功能缺口，服务端 `winner` 与 `report.go`
  都就绪，只差接线。
- `web/src/scene/models/*.js` 是从源项目平移的 ES 模块（未改写逻辑，仅改资产 URL
  与 data 导入）——若视觉对照发现差异，优先查 URL 与 unitData 表。
- 单机模式的 AI 阵营是确定性选取（`enemyFactionOf`，russia 优先），没有难度/随机。
- **交付工具限制（踩了三轮）**：`deliver_task` 无法对 Go 的 `_test.go` 逐文件归因
  （连已提交的 `server_test.go` 都报"目标不存在"），node `.mjs` 测试需先在
  `web/package.json` 加 `test` 脚本才可能被识别。绕过方式经用户确认后使用裸
  `git commit`（`d21c675`、`d6142fe` 两次）。


---

## 11. 第四次交接（2026-10-21）：首次浏览器实拍 + 四个真 bug

### 11.1 实拍验收怎么跑（下个会话可直接复用）

Playwright 在 `/tmp/pw-probe`（本机已装，chromium 复用 `~/Library/Caches/ms-playwright`）。
关键技巧两条，都是踩出来的：

1. **拿战场内部状态**：Vue 的 `setupState` 看不到 `Battle.vue` 里的 `let scene`
   （非响应式，且 `__vueParentComponent` 在自定义元素上取不到链）。
   可靠办法是 **CDP 拦 WS 帧**：
   `ctx.newCDPSession(page)` → `Network.enable` → 监听
   `Network.webSocketFrameReceived`，从 `battle`/`snapshot` 帧直接读 `view`。
2. **判 ghost 颜色别用 WebGL `readPixels`**：`preserveDrawingBuffer=false` 时
   帧外读回全 0。改用 `page.screenshot({clip})` + PNG 解码统计像素。

真实对局跑一遍的耗时基线：开局 6s、发电站建造 17s（含 14s 建造 + 缓冲）。

### 11.2 本轮修掉的四个真 bug（全是照抄源项目时漏的分支）

| # | Bug | 表现 | 根因 | 回归测试 |
| --- | --- | --- | --- | --- |
| 1 | 单机局结算 SIGSEGV | **整个服务进程崩溃**，打完一局就没了 | `end()` 把 AI 席位的 `nil user` 传进 `Settle` 解引用 | `TestSoloEndWithoutAIUser` / `TestSoloEndRecordsNothing` / `TestMultiplayerEndStillSettles` |
| 2 | `resourceSites` 泄漏 Go 字段名 | 前端读 `site.id` 得 `undefined`，所有资源站共用同一退化 key | `gpdata.ResourceSite` 缺 json tag | `TestBattleViewResourceSitesLowercaseKeys` |
| 3 | 单机模式进不了放置态 | 建造完成后没有任何反应，建筑永远放不下 | 源项目读同进程 `game.pendingBuilding`，拆进程后未下发 | 浏览器实拍（`HINT="放置中：power"`） |
| 4 | ghost 恒为绿色 | 非法落点也显示绿色，放置失败无提示 | 源 `render.js:1177` 按 `canPlace` 染红绿，Go 版没做 | `placement.test.mjs` 16/16 + 实拍绿/红色块 |

bug 1 与 §9.4 同族——上次补 `userID()`/`isAI()` 时有**一处漏网**（`end()` 的 Settle 调用）。
教训：**单机 solo 是本项目新增能力，源项目没有**。任何把 `members[i].user` 解引用
传进结算/广播的代码路径都要先判 nil，不能照抄源项目的"成员必有 user"假设。

### 11.3 关于「建筑落位吸附」（上一版遗留判断有误）

实测源项目**没有吸附**：`render.js:1183` 直接把 `this.pointer`（射线与地面交点的
浮点世界坐标）传给 `canPlace`/`placeBuilding`，全仓 grep 无任何放置吸附逻辑。
本轮补的是源项目**有而 Go 版缺**的合法性着色与失败反馈。

### 11.4 新增的协议字段（前端依赖，改 battleView 时别删）

- `view.placeRules`：`{worldWidth, worldHeight, noBuild[], dockZones[], sizes{}}`
  ——源项目 canPlace 在客户端本地跑，拆进程后这些输入必须传输。
- `view.pendingBuilding`：`string | null`，仅单机局非空
  （联机局走 `players[0].readyBuilding`，源 `online-game.js:54`）。

### 11.5 M3 剩余

只剩**与旧版 JS 前端的视觉像素对照**——需要把源项目也跑起来同图同机位截图比对，
或由人/视觉模型目视。功能面（含小地图、框选、资源点、投射物特效、落位着色）已闭合。

### 11.6 交付工具限制（第三次踩到，勿再试）

`deliver_task` 对 Go `_test.go` 的**逐文件归因始终 unsupported**（本轮 4 个 `_test.go`
全部报"目标必须是当前仓库内存在的测试文件"，尽管文件确实存在）。
`run_tests()` 全量跑能拿到可采证记录，但它把 `filter="solo"` 当包名解析 → "setup failed"，
反而污染门禁判成 RED。**结论：Go 侧一律 `go test ./internal/...` + 裸 `git commit`**
（本轮 `4d687c2` 已再次经用户流程确认）。

---

## 12. 第五次交接（2026-10-21）：M3 终局闭环缺失（上一版结论有误）

### 12.1 上一版的错误结论

第四版交接（`436f807`）写的是「M3 功能面完成」。**这是错的**，本轮用浏览器实测证伪。

错因：上一轮我只验到「渲染能出画面、ghost 能红绿」，就把「功能面」当成闭合了。
但重写方案 §六 给 M3 的验收标准是「**新前端连 Go 服务端打完整局**」——
打完一局没有任何结果呈现，就不算完整局。

### 12.2 实测证据（Playwright 操作真实浏览器）

脚本流程：起服务（`-port 4191 -root web/dist`）→ 打开页面 → 点「单机对战 AI」
→ 等 6s → 点「认输退出」→ 观察 DOM 与 WS 帧。

| 观察点 | 实测结果 |
| --- | --- |
| 点击前 | `hasBattle=true`，战场 HUD 正常（资金/电力/时间/部队列表齐全） |
| 点击后 | `hasBattle=false`，`hasLobby=true` |
| 正文内容 | 只有大厅文案，含「战绩 0 胜 / 0 负 / 0 平」 |
| 结算元素 | `.modal` / `.result` / `.settle` **全部不存在** |
| WS 帧序列 | 4×`snapshot`（带 `winner`）→ 3×`room`（`room: null`） |

结论：**不是"停在战场无提示"，而是无提示直接弹回大厅**。
服务端发了一串带 `winner` 的快照，前端一个都没消费，紧接着 `room:null` 把用户踢回大厅。

### 12.3 代码层根因（三处，全是"已实现但没接线"）

| # | 事实 | 位置 |
| --- | --- | --- |
| 1 | 服务端**已下发** `view.winner`（`null`=平局 / `0`=己方胜 / `1`=敌方胜） | `internal/server/view.go:241`，`owner()` 重排在 `:230` |
| 2 | 前端**从未读取** `winner` | `grep -rn "winner" web/src/` → 零命中（除无关的 `assetError`） |
| 3 | 战报包 `report.go` 已完整移植（对齐源 `battle-report.js`），`g.BattleStats` 也在跑，但 `battleView` **从未调用它** | `internal/game/report/report.go` 全包；`grep battleReport internal/server/view.go` → 无 |
| 4 | 源项目**有**这个字段 | `/tmp/gpr-src/server/battle-view.mjs:47` |

即：这是又一处「移植了但没接线」的死代码。`report` 包目前只有 `update.go`/`savegame.go`/
`combat.go` 在累计统计，统计结果无处可去。

### 12.4 修复方案（三步，未实施）

**第 1 步 · 服务端补下发 `battleReport`**（对齐 `battle-view.mjs:47`）

```go
// internal/server/view.go 返回 map 里加：
"battleReport": battleReportView(g, order),
```

语义要点：
- `g.Running` 为真时返回 `nil`（对局进行中无战报）。
- 否则取 `report.Build(g.BattleStats, delivered, intercepted)`，
  再按 `order`（视角方恒为 side 0）重排 `Sides`，`Events[].Owner` 走同一个 `owner()` 映射。
- 源项目还做 `report.sides: order.map(s => report.sides[s])` 与
  `events.map(e => ({...e, owner: owner(e.owner)}))`，两处都要照做。

**第 2 步 · 前端结算层**

`web/src/views/Battle.vue`：
- `const finished = computed(() => v.value?.winner !== undefined && !v.value?.running)`
- 读 `v.winner`：`null` → 平局，`0` → 胜利，`1` → 战败。
- 叠加结算面板：胜负标题 + 双方战损（`battleReport.sides`）+ 关键事件（`events`）
  + 「返回大厅」`net.leave()` / 「再来一局」两个按钮。
- 配色沿用现有 HUD 色板（`#121a24` 底 / `#6ad48f` 正向 / `#e06a5a` 负向），不引入新风格。

**第 3 步 · 验证（唯一能证明"打完整局"的证据）**
- Go：`battleView` 的 `battleReport` 三态测试（进行中 `nil` / 结束非 nil / 侧别重排正确）。
- 前端：若抽出纯函数（胜负文案映射、报表格式化）则加 node 测试。
- 浏览器：真打一局或认输 → 截图确认结算层出现 → 点「返回大厅」能回大厅。

### 12.5 一条方法论教训

**正则命中不等于语义命中。** 本轮第一次跑验收脚本时，我用
`/胜|负|平局/` 判断页面上有没有胜负文案，返回 `true`，差点当成验收通过——
它匹配的是大厅那行「战绩 0 胜 / 0 负 / 0 平」。
改成逐行 dump 正文（`innerText.split('\n')`）才看清真相。

写验收脚本时，断言要锚在**结构**（`.result` 元素是否存在、特定 DOM 节点文本）
而不是**正则扫全文**。
