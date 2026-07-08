# 设计:audit 命令新增「上下架状态」维度

> 日期:2026-07-08
> 状态:设计定稿,待写实现计划
> 关联代码:`pkg/store/audit.go`、`pkg/apkgo/audit.go`、`cmd/audit.go`、各 `pkg/store/<name>/`
> 关联调研:`notes/app-info-update-research.md`、`notes/phased-release-research.md`

## 1. 背景与目标

`apkgo audit` 现在只报告一个**粗粒度审核状态**(`AuditState`:reviewing / approved / rejected / withdrawn / unknown),
它把「上下架」和「审核」两件事压进了同一个枚举里(例如华为「已下架」被映射成 `withdrawn`)。

目标:在现有 `apkgo audit` 上新增一个与「审核状态」**正交**的**上下架状态(listing)**维度,
并把审核维度扩出两个业务需要的新值(`approved_first`、`needs_fix`),覆盖
**华为 / vivo / OPPO / 荣耀 / 应用宝 / 小米** 六家,各尽其能并**如实标注精度**。

主要使用场景(据此定优先级):

- **发布巡检看板**:上传后定期/人工查看各店的「在架 + 审核」状态,一眼看出哪个店过了、被驳回、被下架。要求尽量全渠道覆盖,粗粒度可接受。
- **排查下架 / 驳回**:重点是发现异常(被下架、被驳回、待整改),需要拿到驳回原因 / 下架原因等 detail,渠道越细越好。

## 2. 核心约束(贯穿全设计)

1. **近似纯加法**:不改动现有 review 映射的**骨架**(reviewing / approved / rejected / withdrawn 的判定主体不动)。两个新值是在其上做**细化 / 填补**,可能改变少数场景的输出:
   - `approved_first` 在 `approved` 之上按「有无在架版本」细分 —— 会改变**首次上架**场景的输出(原 `approved` → `approved_first`);
   - `needs_fix` 主要**填补原本落 `unknown`** 的态(如华为 releaseState=9)。
   现有测试中若因这两处细化而受影响的用例,**更新其断言**(改期望值,而非改映射骨架);其余 review 测试保持全绿。
2. **零新增 API 调用**:每家的 listing 都从**该店 auditor 已经在调的那次响应**里额外解码 / 推断,不引入新 endpoint、不新增权限、不改鉴权。
3. **不伪造**:凭现有响应拿不到的态一律 `unknown` + `detail` 说明(区分「能力缺失」与「本次没查到」),**绝不猜 API 字段名 / 硬编码未确认的枚举**。
4. **`--watch` 行为不变**:终止条件仍只看 `review` 是否 resolved,listing 不参与门禁。
5. **向后兼容**:JSON 输出只新增字段,已有字段语义与取值不变。

## 3. 非目标(本次不做)

- 不做「首次 vs 更新通过」的稳态永久区分(无状态查询做不到,除非 apkgo 自持久化历史 —— 明确不做)。
- 不引入任何本地状态存储 / 数据库。
- 不改上传流程,不改 `--watch` 的轮询与终止语义。
- 不为荣耀 / 应用宝 / 小米 先天拿不到的「下架」态强行制造信号。

## 4. 术语:两个正交维度

把用户需求拆成两个**互相独立、各自单一职责**的维度,复合业务态由两者组合 + `detail` 表达:

- **listing(上下架状态)**:应用当前在商店的可见性。
- **review(审核状态)**:最近一次提交 / 版本的审核进展。

## 5. 数据模型改动

### 5.1 `pkg/store/audit.go`

**新增 listing 枚举(4 值):**

```go
// ListingState 是应用在商店的上下架可见性,与 AuditState(审核进展)正交。
type ListingState string

const (
	ListingOnShelf   ListingState = "on_shelf"   // 在架(已发布 / 在售)
	ListingOffShelf  ListingState = "off_shelf"  // 下架(曾在架,已被下架)
	ListingNotListed ListingState = "not_listed" // 未上架(草稿 / 从未上架 / 首次待审)
	ListingUnknown   ListingState = "unknown"    // 该店无接口 或 本次拿不到 —— detail 说明原因
)
```

**扩展 review 枚举(5 → 7 值,只加不改):**

```go
const (
	AuditReviewing AuditState = "reviewing" // 审核中(不变)
	AuditApproved  AuditState = "approved"  // 审核通过 / 更新通过(不变)
	AuditRejected  AuditState = "rejected"  // 审核驳回(不变)
	AuditWithdrawn AuditState = "withdrawn" // 已撤回(不变)
	AuditUnknown   AuditState = "unknown"   // 未映射(不变)

	// 新增:
	AuditApprovedFirst AuditState = "approved_first" // 审核通过-首次上架(通过/待发布 且 无在架版本)
	AuditNeedsFix      AuditState = "needs_fix"      // 待整改(原始标签暴露"整改"语义时才点亮)
)
```

> `AuditState.Resolved()` 需要把两个新值纳入「终态」判断:`approved_first` 视为已解决(等同 approved);
> `needs_fix` 视为**未解决**(等同还在处理,`--watch` 应继续轮询直到它变成 approved/rejected/withdrawn)。
> 具体:`Resolved()` 返回 true 的集合 = {approved, approved_first, rejected, withdrawn}。needs_fix、reviewing、unknown 不算终态。

**`AuditResult` 新增一个字段:**

```go
type AuditResult struct {
	Store  string     `json:"store"`
	State  AuditState `json:"state,omitempty"`   // review 维度(不变)
	Detail string     `json:"detail,omitempty"`  // 原始标签 / 驳回原因 / 上下架补充说明(不变,承载更多信息)
	Error  string     `json:"error,omitempty"`

	Listing ListingState `json:"listing,omitempty"` // 新增:上下架维度

	VersionName     string `json:"version_name,omitempty"`
	VersionCode     int32  `json:"version_code,omitempty"`
	LiveVersionName string `json:"live_version_name,omitempty"`
	LiveVersionCode int32  `json:"live_version_code,omitempty"`
}
```

- **不新增 `listing_detail` 字段**:上下架的补充说明统一并进已有的 `detail`(避免结构膨胀)。
- 华为把 raw `releaseState` 数值一并写进 `detail`(如 `已下架(releaseState=2)`),排查时信息不丢。

### 5.2 `pkg/apkgo/audit.go`

`AuditStoreResult` 内嵌 `store.AuditResult`,自动获得 `Listing` 字段,**无需改动结构**。
`AllResolved()` 已基于 `s.State.Resolved()`,只要 `Resolved()` 正确纳入新值即可,无需改 `AllResolved` 本身。

## 6. 复合业务态 = 两维度组合

用户列的业务态由 `listing` + `review`(+ `detail`)组合表达,不再是单一枚举:

| 业务态 | listing | review | detail 举例 |
|---|---|---|---|
| 上架审核通过(首次) | `on_shelf` / `not_listed` | `approved_first` | 华为 raw releaseState |
| 上架审核驳回 | `not_listed` | `rejected` | 驳回原因 |
| 审核中 | (原样) | `reviewing` | |
| 待整改 | (原样) | `needs_fix` | "待整改" / "下架审核不通过" |
| 下架 | `off_shelf` | (原样,常为 approved) | |
| 下架整改 | `off_shelf` | `needs_fix` | 用户定义的组合 |
| 更新通过 | `on_shelf` | `approved` | 有在架版本时的通过 |
| 更新驳回 | `on_shelf` | `rejected` | "升级审核不通过" |

关键原则:每个维度单一职责,「下架整改」这类复合态**不做成单独枚举值**,而是 `off_shelf` + `needs_fix` 的组合。

## 7. 各渠道推导规则(核心)

全部复用 auditor 已有的那一次 API 响应,零新增调用。

### 7.1 华为 huawei —— 精度最高

- **数据源**:`GET /api/publish/v2/app-info`(`appId` + `releaseState`、`onShelfVersionCode`),现有 `mapHuaweiReleaseState`(`huawei.go:115-130`)。
- **releaseState 全枚举**(`huawei.go:112-114`):0 已上架 / 1 上架审核不通过 / 2 已下架 / 3 待上架(预约) / 4 审核中 / 5 升级审核中 / 6 申请下架 / 7 草稿 / 8 升级审核不通过 / 9 下架审核不通过 / 10 开发者下架 / 11 撤销上架 / 12 预审中 / 13 预审不通过。
- **review 映射**:保持 `mapHuaweiReleaseState` 现有输出**不动**(4/5/12→reviewing、0/3→approved、1/8/13→rejected、2/10/11→withdrawn、7→unknown、default→unknown),然后**叠加**新值的判定:
  - `approved_first`:当映射结果为 `approved`(releaseState ∈ {0,3})**且** `onShelfVersionCode == 0`(从无在架版本)→ 覆写为 `approved_first`;`onShelfVersionCode > 0` → 保持 `approved`(即更新通过)。
  - `needs_fix`:releaseState == 9(下架审核不通过)→ `needs_fix`(当前 default 落 unknown,改为 needs_fix)。releaseState == 6(申请下架)保持 unknown 或按实现阶段确认。
- **listing 映射**(新增,独立于 review):
  - 0 → `on_shelf`
  - 2 / 6 / 9 / 10 / 11 → `off_shelf`
  - 7 / 1 / 13 → `not_listed`(草稿 / 首次上架未过)
  - 3 → `not_listed`(待上架/预约,尚未在架;详见 §10 开放项)
  - 4 / 5 / 8 / 12(审核 / 升级 / 预审期)→ `onShelfVersionCode > 0 ? on_shelf : not_listed`(有在架版本说明是更新在审,应用仍在架)
- **detail**:附带 raw releaseState,如 `已下架(releaseState=2)`。
- 参数:`appId`(缺省用包名调 `GET /api/publish/v2/appid-list` 自动解析,`huawei.go:312-346`)。

### 7.2 OPPO —— 关键词软匹配

- **数据源**:`GET /resource/v1/app/info`(`pkg_name`)→ `audit_status_name`(中文标签)、`audit_status`(数字,码表未公开)、`refuse_reason`。现有 `mapOppoAudit`(`oppo.go:74-92`)。
- **review**:保持现有关键词映射不动,**新增** `needs_fix`:`audit_status_name` 含「整改」→ `needs_fix`(优先于其它关键词)。
- **listing**(新增,基于同一 `audit_status_name`):
  - 含「上线 / 上架 / 已发布 / 在架」→ `on_shelf`
  - 含「下架 / 冻结 / 撤销」→ `off_shelf`
  - 否则 → `unknown`
- **approved_first**:OPPO 无独立在架历史版本字段 → best-effort;拿不到则退回 `approved` / `unknown`。
- **实现阶段尝试**:解码数字 `audit_status`(`oppo.go:578` 已定义但未用)以获得更稳的枚举;码表查不到就维持关键词方案(见 §10)。

### 7.3 vivo —— 需核实在架字段

- **数据源**:`POST app.query.details`(`packageName`)→ `status`(1 草稿 / 2 待审核 / 3 通过 / 4 不通过 / 5 撤销),现有 `mapVivoAuditState`(`vivo.go:75-90`)。返回体注释(`vivo.go:537-539`)称还含「online state」但**未解码**。
- **review**:保持现有 5 态映射不动。vivo 无「整改」态 → 不产 `needs_fix`。
- **listing**(新增):解码 `app.query.details` 的在架状态字段 →`on_shelf` / `off_shelf`。**字段名 / 枚举值实现阶段查官方文档确认**(见 §10);核不出来则 listing 降级 `unknown` + detail 说明。
- **approved_first**:vivo 无独立在架历史版本 → 退回 `approved` / `unknown`。

### 7.4 小米 xiaomi —— 只两态,识别不了下架

- **数据源**:`POST /dev/query`(`packageName`)→ `packageInfo`(仅版本号,无状态字段)。现有 `audit()` 靠「提交版本 vs 线上版本」推断(`xiaomi.go:66-106`)。
- **review**:保持现有推断不动(info==nil 且 VersionCode>0 → reviewing;submitted>live → reviewing;live>=submitted → approved)。**无法产** rejected / needs_fix。
- **listing**(新增):
  - `info == nil`(账号下无此包)→ `not_listed`
  - `info != nil`(有在架版本)→ `on_shelf`
  - **无法识别 `off_shelf`**(下架应用是否仍返回 packageInfo 未知,见 §10)→ 不制造该态。
- **approved_first**:小米在稳态下**基本无法判定「首次」**(查到在架版本时无从知道此前是否有过在架版本)。仅当 `info == nil` 但 `AuditQuery.VersionCode > 0`(首次提交、尚无在架版本)这一路可视为「首次通过前/审核中」,通常仍落 `reviewing` 而非 `approved_first`。故小米 `approved_first` 多数退回 `approved`,如实标注为弱能力。

### 7.5 应用宝 tencent —— 有审核态,上下架靠推断

- **数据源**:`POST /query_app_update_status`(`pkg_name` + `app_id`)→ `audit_status`(0 无提交 / 1 审核中 / 2 驳回 / 3 通过 / 8 撤回)、`audit_reason`;+ 公开详情页 `https://sj.qq.com/appdetail/{pkg}` 抓 `LiveVersionName`(仅版本名)。现有 `audit()`(`tencent.go:371-402`)。
- **review**:保持现有映射不动。应用宝无「整改」态 → 不产 `needs_fix`。
- **listing**(新增):
  - 公开页抓到该包版本(`LiveVersionName != ""`)→ `on_shelf`
  - `query_app_detail` 绑定报错 / 公开页抓不到 → `not_listed`
  - **无法可靠识别 `off_shelf`** → 不制造该态。
- **approved_first**:`audit_status == 3`(通过)且 `LiveVersionName == ""`(公开页无在架版本)→ `approved_first`;否则 `approved`。
- 参数:`app_id` 必填(`app_id` / `app_id_map`,无 listing API 无法自动发现)。

### 7.6 荣耀 honor —— 上下架不支持

- **数据源**:审核态 `POST /openapi/v1/publish/get-audit-result`(需 `appId` + `releaseId`,`releaseId` 仅「经 apkgo 上传过」的版本才有);无 releaseId 时退化到 `GET /openapi/v1/publish/get-app-detail` 仅拿线上版本。现有实现 `honor.go:60-158`。
- **review**:保持现有映射不动(0→reviewing、1→approved、2→rejected、3/4→unknown)。荣耀无「整改」态 → 不产 `needs_fix`。
- **listing**(新增):荣耀**无可靠上下架接口**(`get-app-current-release.releaseStatus` 未使用且注释判定不可靠)→ 恒为 `ListingUnknown` + detail「荣耀无上下架查询接口」。
- **approved_first**:`get-app-detail` 返回的 `releaseInfo` 为空 / 无线上版本,且审核态为 approved(有 releaseId 时)→ best-effort `approved_first`;否则 `approved`。

### 7.7 汇总:各店可达精度

| 渠道 | listing | review 细度 | approved_first | needs_fix |
|---|---|---|---|---|
| 华为 | 精确(releaseState) | 全 | ✅ onShelfVersionCode==0 | ✅ releaseState=9 |
| OPPO | 关键词(上线/下架) | 中 | best-effort/unknown | ✅ 关键词「整改」 |
| vivo | online-state 字段(待核) | 5 态 | 退回 approved/unknown | ❌ 无此态 |
| 小米 | 在架 / 未上架(识别不了下架) | reviewing/approved | best-effort | ❌ |
| 应用宝 | 未上架推断(识别不了下架) | 5 态 | ✅ 靠 LiveVersion 抓取为空 | ❌ |
| 荣耀 | unknown(无接口) | 需 releaseId,3 态 | best-effort(get-app-detail) | ❌ |

## 8. 命令与输出

命令、参数**不变**:`apkgo audit -p <pkg> [-f] [-s stores] [--watch] [--interval] [-o json|text]`。

### 8.1 JSON

每个 store 段新增 `listing` 字段:

```json
{
  "package": "com.example.app",
  "stores": [
    { "store": "huawei", "listing": "on_shelf", "state": "approved",
      "detail": "已上架(releaseState=0)", "live_version_name": "1.2.0", "live_version_code": 120, "supported": true },
    { "store": "xiaomi", "listing": "unknown", "state": "reviewing",
      "detail": "提交版本 121 高于在架版本 120,等待上线", "live_version_code": 120, "supported": true }
  ]
}
```

### 8.2 text(`-o text`)

`renderAuditText` / `auditOneLiner`(`cmd/audit.go`)每行新增一列上下架图标 + 文案,例如:

```
huawei   🟢 在架     ✅ approved (已上架 releaseState=0, v1.2.0)
xiaomi   ❔ 未知     ⏳ reviewing (提交版本高于在架版本)
honor    ❔ 未知     ✅ approved (releaseId=...)
```

- 新增 `listingGlyph(state)` 映射:`on_shelf`→🟢在架 / `off_shelf`→🔴下架 / `not_listed`→⚪未上架 / `unknown`→❔未知,配 ANSI 颜色(复用 `colorize`)。
- `auditGlyph` 需为新值补图标:`approved_first`→🎉、`needs_fix`→🛠。
- `--watch` 进度行(stderr,`renderAuditProgress`)保持只显示 review 态,不受影响。

## 9. 错误处理与降级

- 单店查询失败仍走各自 `AuditResult.Error`,`Listing` 留空(omitempty 不输出),整份报告照常产出(现有行为不变)。
- vivo online-state 字段 / OPPO 数字码表若实现阶段核不到:该店 listing 降级 `ListingUnknown` + detail 说明,不猜字段名。
- 荣耀 / 小米 / 应用宝 先天拿不到的态,统一 `ListingUnknown` + detail,不伪造。

## 10. 实现阶段需核实的开放项(查不到就按降级方案)

1. **vivo** `app.query.details` 响应里「在架状态」的确切**字段名与枚举值**(查 vivo 开放平台官方文档 / 抓真实响应)。核不到 → vivo listing 恒 `unknown`。
2. **OPPO** `audit_status` 数字码表(`oppo.go:578` 有字段无码表)。拿不到 → 继续用 `audit_status_name` 关键词。
3. **华为** `releaseState=3`(待上架/预约)归 `not_listed` 还是 `on_shelf`——本设计暂定 `not_listed` + detail 标注;若发现「预约」多指已在架的定时更新,再调整。
4. **华为** `releaseState=6`(申请下架)review 归属(unknown vs needs_fix vs 保持 withdrawn 语义)。
5. **小米 / 应用宝** 下架应用的响应形态:能否借此识别 `off_shelf`;查不到就维持「识别不了下架」的诚实结论。

## 11. 测试策略

- 每店加 table-driven 测试,用假 HTTP 响应(沿用各店现有测试的 fixture 风格,如 `huawei/audit_test.go`、`tencent/tencent_test.go`)覆盖 `on_shelf / off_shelf / not_listed / unknown` 各分支。
- **华为**重点覆盖 14 个 releaseState → (listing, review) 的映射矩阵,含 `approved_first`(onShelfVersionCode==0 与 >0 两路)。
- 现有 review 测试:除被 `approved_first` / `needs_fix` 细化直接影响的少数用例(更新其期望值)外,其余保持全绿,证明未破坏映射骨架。
- `pkg/store/audit_test.go` 补 `Resolved()` 对 `approved_first`(true)/ `needs_fix`(false)的断言。
- `cmd` 层:JSON 含 `listing` 字段、text 渲染新列的快照 / 断言。

## 12. 文件改动清单(预估)

| 文件 | 改动 |
|---|---|
| `pkg/store/audit.go` | 新增 `ListingState` 枚举 + `AuditResult.Listing` 字段 + `AuditState` 两个新值 + `Resolved()` 纳入新值 |
| `pkg/store/huawei/huawei.go` | listing 映射 + approved_first/needs_fix 叠加 |
| `pkg/store/oppo/oppo.go` | listing 关键词 + needs_fix 关键词(可选:数字码表) |
| `pkg/store/vivo/vivo.go` | 解码 online-state → listing(待核) |
| `pkg/store/xiaomi/xiaomi.go` | listing 推断(在架/未上架)+ approved_first best-effort |
| `pkg/store/tencent/tencent.go` | listing 推断 + approved_first |
| `pkg/store/honor/honor.go` | listing 恒 unknown + approved_first best-effort |
| `cmd/audit.go` | `listingGlyph` + text 新列 + `auditGlyph` 补新值图标 |
| 各 `*_test.go` | listing / 新 review 值的 table-driven 测试 |
| `pkg/apkgo/audit.go` | 无需改结构(内嵌自动获得 Listing) |
| `CLAUDE.md` | 更新 `apkgo audit` 段落,说明 listing 维度与新增 review 值 |

## 13. 交付顺序建议(供写实现计划参考)

1. 类型层(§5)+ `Resolved()` + 测试 —— 地基,不碰任何店。
2. 华为(精度最高、信号最全,验证整套模型)+ 测试。
3. OPPO / 小米 / 应用宝 / 荣耀(现有响应即可推断,无外部依赖)+ 测试。
4. vivo(依赖官方文档核实在架字段,最后做,核不到就降级)+ 测试。
5. cmd 渲染 + JSON + CLAUDE.md 文档。
