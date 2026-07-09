# audit 命令新增「上下架状态」维度 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `apkgo audit` 上正交新增 `listing`(上下架)维度,并把 `review` 枚举扩出 `approved_first`(首次上架通过)与 `needs_fix`(待整改)两个值,覆盖华为/OPPO/vivo/小米/应用宝/荣耀六家。

**Architecture:** 纯加法扩展。在 `pkg/store/audit.go` 新增 `ListingState` 枚举与 `AuditResult.Listing` 字段;每个 store 的 auditor 从它**已经在调**的那次 API 响应里额外推导 listing(零新增 endpoint);渲染层(`cmd/audit.go`)在每行前面加一列上下架状态。所有可测逻辑抽成纯函数,沿用各 store 现有「只测纯映射函数」的测试风格。

**Tech Stack:** Go(module `github.com/KevinGong2013/apkgo/v3`)、resty HTTP、标准 `go test`。

## Global Constraints

- 沟通/文档/注释/commit message 用简体中文;代码标识符用英文(项目既有约定)。
- **近似纯加法**:不改动 review 映射骨架(reviewing/approved/rejected/withdrawn 判定主体不动)。`approved_first` 在 `approved` 上按「有无在架版本」细化(会改变首次上架场景输出);`needs_fix` 主要填补原本落 `unknown` 的态。受这两处影响的既有测试用例更新其期望值,其余保持全绿。
- **零新增 API 调用**:每家 listing 只从该店 auditor 已有的那次响应推导,不加 endpoint、不改鉴权、不加权限。
- **不伪造**:拿不到的态一律 `ListingUnknown` + `detail` 说明,绝不猜 API 字段名/硬编码未确认枚举而产出错误数据。
- **`--watch` 语义**:`AuditState.Resolved()` 决定终止;`approved_first` 记为终态,`needs_fix` 记为未终态(继续轮询)。
- **向后兼容**:JSON 只新增字段(`listing`,omitempty),既有字段语义/取值不变。
- 每个 store 的 auditor 签名固定为 `func audit(ctx context.Context, cfg map[string]string, q store.AuditQuery) store.AuditResult`,不改。

---

## 文件结构

| 文件 | 职责 | 本计划改动 |
|---|---|---|
| `pkg/store/audit.go` | 统一 audit 抽象/枚举 | 新增 `ListingState` + `AuditResult.Listing` + 两个 `AuditState` 值 + `Resolved()` |
| `pkg/store/audit_test.go` | `Resolved()` 契约测试 | 补 `approved_first`/`needs_fix` 断言 |
| `pkg/store/huawei/huawei.go` | 华为 auditor | `mapHuaweiListing`、`reviewFromReleaseState`、`mapHuaweiReleaseState` 加 case 9、wire |
| `pkg/store/huawei/audit_test.go` | 华为映射测试 | 补 listing + approved_first + case 9 表 |
| `pkg/store/oppo/oppo.go` | OPPO auditor | `oppoListing`、`mapOppoAudit` 加「整改」、wire |
| `pkg/store/oppo/oppo_test.go`(新建) | OPPO 映射测试 | listing + needs_fix 表 |
| `pkg/store/xiaomi/xiaomi.go` | 小米 auditor | `xiaomiListing`、wire |
| `pkg/store/xiaomi/xiaomi_test.go`(新建) | 小米映射测试 | listing 表 |
| `pkg/store/tencent/tencent.go` | 应用宝 auditor | `tencentListing`、`applyTencentFirstListing`、wire |
| `pkg/store/tencent/tencent_test.go` | 应用宝测试 | listing + approved_first 表 |
| `pkg/store/honor/honor.go` | 荣耀 auditor | listing 恒 `unknown`、wire |
| `pkg/store/vivo/vivo.go` | vivo auditor | `vivoListing` + online-state 字段(需核实)、wire |
| `pkg/store/vivo/vivo_test.go`(新建) | vivo 映射测试 | listing 表 |
| `cmd/audit.go` | 渲染层 | `listingGlyph`、`auditOneLiner` 加列、`auditGlyph` 补新值图标 |
| `CLAUDE.md` | 项目文档 | 更新 audit 段落 |

---

## Task 1: 类型层 —— ListingState 枚举 + AuditResult.Listing + 两个 review 新值

**Files:**
- Modify: `pkg/store/audit.go`
- Test: `pkg/store/audit_test.go`

**Interfaces:**
- Produces: `store.ListingState`(`ListingOnShelf`/`ListingOffShelf`/`ListingNotListed`/`ListingUnknown`);`store.AuditResult.Listing ListingState`;`store.AuditApprovedFirst`/`store.AuditNeedsFix`;`AuditState.Resolved()` 纳入 `approved_first`(终态)、不纳入 `needs_fix`。

- [ ] **Step 1: 写失败测试(Resolved 对新值的契约)**

在 `pkg/store/audit_test.go` 的 `TestAuditStateResolved` 里,把新值加入两组断言:

```go
func TestAuditStateResolved(t *testing.T) {
	resolved := []store.AuditState{
		store.AuditApproved, store.AuditRejected, store.AuditWithdrawn,
		store.AuditApprovedFirst, // 首次上架通过属终态
	}
	for _, s := range resolved {
		if !s.Resolved() {
			t.Errorf("%q.Resolved() = false, want true", s)
		}
	}
	pending := []store.AuditState{
		store.AuditReviewing, store.AuditUnknown, store.AuditState(""),
		store.AuditNeedsFix, // 待整改仍需继续轮询
	}
	for _, s := range pending {
		if s.Resolved() {
			t.Errorf("%q.Resolved() = true, want false", s)
		}
	}
}
```

- [ ] **Step 2: 运行测试,确认编译失败**

Run: `go test ./pkg/store/ -run TestAuditStateResolved`
Expected: 编译错误 `undefined: store.AuditApprovedFirst` / `store.AuditNeedsFix`。

- [ ] **Step 3: 加两个 AuditState 值并更新 Resolved()**

在 `pkg/store/audit.go` 的 `const (...)` 块(现有 `AuditUnknown` 之后)追加:

```go
	// AuditApprovedFirst 审核通过-首次上架:通过/待发布 且 该应用尚无在架版本
	// (live version 为空)。用于把"首次上架通过"与"更新通过"区分开。
	AuditApprovedFirst AuditState = "approved_first"
	// AuditNeedsFix 待整改:仅在渠道原始标签/状态确实暴露"整改"语义时点亮。
	AuditNeedsFix AuditState = "needs_fix"
```

更新 `Resolved()`:

```go
func (s AuditState) Resolved() bool {
	return s == AuditApproved || s == AuditApprovedFirst ||
		s == AuditRejected || s == AuditWithdrawn
}
```

- [ ] **Step 4: 加 ListingState 枚举 + AuditResult.Listing 字段**

在 `pkg/store/audit.go` 里 `AuditState` 定义之后新增:

```go
// ListingState 是应用在商店的上下架可见性,与 AuditState(审核进展)正交。
type ListingState string

const (
	ListingOnShelf   ListingState = "on_shelf"   // 在架(已发布/在售)
	ListingOffShelf  ListingState = "off_shelf"  // 下架(曾在架,已被下架)
	ListingNotListed ListingState = "not_listed" // 未上架(草稿/从未上架/首次待审)
	ListingUnknown   ListingState = "unknown"    // 该店无接口 或 本次拿不到 —— detail 说明原因
)
```

在 `AuditResult` 结构体里,`State` 字段之后新增(`Detail` 之前或之后均可,保持 json 顺序清晰):

```go
	// Listing 是与 State 正交的上下架维度。为空表示该店未产出(omitempty 不输出);
	// 明确的 ListingUnknown 表示"该店无此能力/本次拿不到",detail 说明原因。
	Listing ListingState `json:"listing,omitempty"`
```

- [ ] **Step 5: 运行测试,确认通过**

Run: `go test ./pkg/store/ -run TestAuditStateResolved`
Expected: PASS。

- [ ] **Step 6: 确认全仓编译通过**

Run: `go build ./...`
Expected: 无输出(成功)。新字段不影响既有代码。

- [ ] **Step 7: 提交**

```bash
git add pkg/store/audit.go pkg/store/audit_test.go
git commit -m "feat(audit): 新增 ListingState 枚举与 approved_first/needs_fix 审核态"
```

---

## Task 2: 华为 —— listing 精确映射 + approved_first + needs_fix(case 9)

**Files:**
- Modify: `pkg/store/huawei/huawei.go`(`audit()` 约 `huawei.go:103-107`、`mapHuaweiReleaseState` 约 `:115-130`)
- Test: `pkg/store/huawei/audit_test.go`

**Interfaces:**
- Consumes: Task 1 的 `store.ListingState`、`store.AuditApprovedFirst`、`store.AuditNeedsFix`。
- Produces: `mapHuaweiListing(state int, onShelfVersionCode int64) store.ListingState`;`reviewFromReleaseState(state int, onShelfVersionCode int64) (store.AuditState, string)`。

- [ ] **Step 1: 写失败测试(listing 映射矩阵)**

在 `pkg/store/huawei/audit_test.go` 追加:

```go
func TestMapHuaweiListing(t *testing.T) {
	cases := []struct {
		state      int
		onShelf    int64
		want       store.ListingState
	}{
		{0, 100, store.ListingOnShelf},   // 已上架
		{2, 100, store.ListingOffShelf},  // 已下架
		{6, 100, store.ListingOffShelf},  // 申请下架
		{9, 100, store.ListingOffShelf},  // 下架审核不通过
		{10, 100, store.ListingOffShelf}, // 开发者下架
		{11, 100, store.ListingOffShelf}, // 撤销上架
		{7, 0, store.ListingNotListed},   // 草稿
		{1, 0, store.ListingNotListed},   // 上架审核不通过(首次)
		{13, 0, store.ListingNotListed},  // 预审不通过
		{3, 0, store.ListingNotListed},   // 待上架/预约,尚未在架
		{4, 100, store.ListingOnShelf},   // 审核中 + 有在架版本 → 更新在审,仍在架
		{4, 0, store.ListingNotListed},   // 审核中 + 无在架版本 → 首次在审
		{5, 100, store.ListingOnShelf},   // 升级审核中 + 有在架版本
		{8, 100, store.ListingOnShelf},   // 升级审核不通过 + 有在架版本(旧版仍在架)
		{99, 0, store.ListingUnknown},    // 未知
	}
	for _, tc := range cases {
		if got := mapHuaweiListing(tc.state, tc.onShelf); got != tc.want {
			t.Errorf("mapHuaweiListing(%d, %d) = %q, want %q", tc.state, tc.onShelf, got, tc.want)
		}
	}
}

func TestReviewFromReleaseState(t *testing.T) {
	cases := []struct {
		state   int
		onShelf int64
		want    store.AuditState
	}{
		{0, 0, store.AuditApprovedFirst}, // 通过且无在架版本 → 首次上架通过
		{3, 0, store.AuditApprovedFirst}, // 待上架且无在架版本 → 首次上架通过
		{0, 120, store.AuditApproved},    // 通过且已有在架版本 → 更新通过
		{4, 0, store.AuditReviewing},     // 审核中不受影响
		{9, 0, store.AuditNeedsFix},      // 下架审核不通过 → 待整改
		{1, 0, store.AuditRejected},      // 上架审核不通过
	}
	for _, tc := range cases {
		if got, _ := reviewFromReleaseState(tc.state, tc.onShelf); got != tc.want {
			t.Errorf("reviewFromReleaseState(%d, %d) = %q, want %q", tc.state, tc.onShelf, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: 运行测试,确认失败**

Run: `go test ./pkg/store/huawei/ -run 'TestMapHuaweiListing|TestReviewFromReleaseState'`
Expected: 编译错误 `undefined: mapHuaweiListing` / `reviewFromReleaseState`。

- [ ] **Step 3: 加 mapHuaweiListing + reviewFromReleaseState,并给 mapHuaweiReleaseState 加 case 9**

在 `pkg/store/huawei/huawei.go` 的 `mapHuaweiReleaseState` 里,`case 2, 10, 11` 之后新增 case 9(原先落 default→unknown,现填成 needs_fix):

```go
	case 9:
		return store.AuditNeedsFix, "下架审核不通过(releaseState=9)"
```

在 `mapHuaweiReleaseState` 之后新增两个函数:

```go
// reviewFromReleaseState 在 mapHuaweiReleaseState 之上叠加"首次上架通过"细化:
// 当审核通过/待发布(0/3)且该应用尚无在架版本(onShelfVersionCode==0)时,
// 记为 approved_first;有在架版本则是更新通过(approved)。
func reviewFromReleaseState(state int, onShelfVersionCode int64) (store.AuditState, string) {
	st, detail := mapHuaweiReleaseState(state)
	if st == store.AuditApproved && onShelfVersionCode == 0 {
		st = store.AuditApprovedFirst
	}
	return st, detail
}

// mapHuaweiListing 把 releaseState 映射到与审核正交的上下架维度。审核期
// (4/5/8/12)靠 onShelfVersionCode 判断:有在架版本说明是更新在审,应用仍在架。
func mapHuaweiListing(state int, onShelfVersionCode int64) store.ListingState {
	switch state {
	case 0:
		return store.ListingOnShelf
	case 2, 6, 9, 10, 11:
		return store.ListingOffShelf
	case 1, 3, 7, 13:
		return store.ListingNotListed
	case 4, 5, 8, 12:
		if onShelfVersionCode > 0 {
			return store.ListingOnShelf
		}
		return store.ListingNotListed
	default:
		return store.ListingUnknown
	}
}
```

- [ ] **Step 4: wire 进 audit()**

在 `pkg/store/huawei/huawei.go` 的 `audit()` 里,把:

```go
	res.State, res.Detail = mapHuaweiReleaseState(af.ReleaseState)
	res.VersionName = af.VersionNumber
```

改为:

```go
	res.State, res.Detail = reviewFromReleaseState(af.ReleaseState, af.OnShelfVersionCode)
	res.Listing = mapHuaweiListing(af.ReleaseState, af.OnShelfVersionCode)
	res.VersionName = af.VersionNumber
```

- [ ] **Step 5: 运行测试,确认新测试通过、既有测试仍绿**

Run: `go test ./pkg/store/huawei/`
Expected: PASS(含既有 `TestMapHuaweiReleaseState` —— 它未测 case 9,不受影响)。

- [ ] **Step 6: 提交**

```bash
git add pkg/store/huawei/huawei.go pkg/store/huawei/audit_test.go
git commit -m "feat(huawei): audit 新增上下架维度与首次上架/待整改细化"
```

---

## Task 3: OPPO —— listing 关键词 + needs_fix(整改)

**Files:**
- Modify: `pkg/store/oppo/oppo.go`(`audit()` 约 `oppo.go:61`、`mapOppoAudit` 约 `:74-92`)
- Test(新建): `pkg/store/oppo/oppo_test.go`

**Interfaces:**
- Consumes: Task 1 的 `store.ListingState`、`store.AuditNeedsFix`;既有 `containsAny`(`oppo.go:94`)。
- Produces: `oppoListing(name string) store.ListingState`;`mapOppoAudit` 增加「整改」→ needs_fix 分支。

- [ ] **Step 1: 写失败测试(新建 oppo_test.go)**

创建 `pkg/store/oppo/oppo_test.go`:

```go
package oppo

import (
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

func TestOppoListing(t *testing.T) {
	cases := map[string]store.ListingState{
		"已上线":     store.ListingOnShelf,
		"上架成功":    store.ListingOnShelf,
		"已发布":     store.ListingOnShelf,
		"已下架":     store.ListingOffShelf,
		"已冻结":     store.ListingOffShelf,
		"审核中":     store.ListingUnknown,
		"":        store.ListingUnknown,
	}
	for name, want := range cases {
		if got := oppoListing(name); got != want {
			t.Errorf("oppoListing(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestMapOppoAuditNeedsFix(t *testing.T) {
	got, _ := mapOppoAudit("待整改", "")
	if got != store.AuditNeedsFix {
		t.Errorf("mapOppoAudit(待整改) = %q, want %q", got, store.AuditNeedsFix)
	}
	// 既有关键词不受影响
	if got, _ := mapOppoAudit("已上线", ""); got != store.AuditApproved {
		t.Errorf("mapOppoAudit(已上线) = %q, want approved", got)
	}
}
```

- [ ] **Step 2: 运行测试,确认失败**

Run: `go test ./pkg/store/oppo/ -run 'TestOppoListing|TestMapOppoAuditNeedsFix'`
Expected: 编译错误 `undefined: oppoListing`。

- [ ] **Step 3: 加 oppoListing,并给 mapOppoAudit 加「整改」分支**

在 `pkg/store/oppo/oppo.go` 的 `mapOppoAudit` 里,`switch` 的**第一个 case 之前**插入(整改优先级最高,避免被"不通过"等词吞掉):

```go
	case containsAny(name, "整改"):
		return store.AuditNeedsFix, name
```

在 `mapOppoAudit` 之后新增:

```go
// oppoListing 从 app/info 的 audit_status_name 关键词推导上下架维度。
// OPPO 未公开数字码表,故与审核映射一样走关键词软匹配;拿不到即 unknown。
func oppoListing(name string) store.ListingState {
	switch {
	case containsAny(name, "上线", "上架", "已发布", "在架"):
		return store.ListingOnShelf
	case containsAny(name, "下架", "冻结", "撤销"):
		return store.ListingOffShelf
	default:
		return store.ListingUnknown
	}
}
```

- [ ] **Step 4: wire 进 audit()**

在 `pkg/store/oppo/oppo.go` 的 `audit()` 里,把:

```go
	res.State, res.Detail = mapOppoAudit(app.AuditStatusName, app.RefuseReason)
```

之后加一行:

```go
	res.Listing = oppoListing(app.AuditStatusName)
```

- [ ] **Step 5: 运行测试,确认通过**

Run: `go test ./pkg/store/oppo/`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add pkg/store/oppo/oppo.go pkg/store/oppo/oppo_test.go
git commit -m "feat(oppo): audit 新增上下架维度与待整改识别"
```

---

## Task 4: 小米 —— listing 在架/未上架推断

**Files:**
- Modify: `pkg/store/xiaomi/xiaomi.go`(`audit()` 约 `xiaomi.go:66-106`)
- Test(新建): `pkg/store/xiaomi/xiaomi_test.go`

**Interfaces:**
- Consumes: Task 1 的 `store.ListingState`。
- Produces: `xiaomiListing(listed bool) store.ListingState`。

小米无审核状态接口,`/dev/query` 只返回版本号。listing 只能判「有在架版本(on_shelf)」vs「账号下无此包(not_listed)」,**无法识别 off_shelf**(下架应用是否仍返回 packageInfo 未知,见 spec §10.5)。review 维度保持现有推断不动;小米稳态下无法可靠判「首次」,故不产 approved_first。

- [ ] **Step 1: 写失败测试(新建 xiaomi_test.go)**

创建 `pkg/store/xiaomi/xiaomi_test.go`:

```go
package xiaomi

import (
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

func TestXiaomiListing(t *testing.T) {
	if got := xiaomiListing(true); got != store.ListingOnShelf {
		t.Errorf("xiaomiListing(true) = %q, want on_shelf", got)
	}
	if got := xiaomiListing(false); got != store.ListingNotListed {
		t.Errorf("xiaomiListing(false) = %q, want not_listed", got)
	}
}
```

- [ ] **Step 2: 运行测试,确认失败**

Run: `go test ./pkg/store/xiaomi/ -run TestXiaomiListing`
Expected: 编译错误 `undefined: xiaomiListing`。

- [ ] **Step 3: 加 xiaomiListing**

在 `pkg/store/xiaomi/xiaomi.go` 的 `audit()` 之后新增:

```go
// xiaomiListing 由 /dev/query 是否返回 packageInfo 推断上下架:有在架版本→
// on_shelf,账号下无此包→not_listed。小米无法识别"下架"(见 spec §10.5)。
func xiaomiListing(listed bool) store.ListingState {
	if listed {
		return store.ListingOnShelf
	}
	return store.ListingNotListed
}
```

- [ ] **Step 4: wire 进 audit()**

在 `pkg/store/xiaomi/xiaomi.go` 的 `audit()` 里,`info == nil` 分支内(`res.State = store.AuditUnknown` / `AuditReviewing` 那段)与非 nil 分支各补一行 listing。最简做法:在 `info, err := s.query(q.Package)` 错误处理之后、`const inferred` 之前统一加:

```go
	res.Listing = xiaomiListing(info != nil)
```

(该行在两个分支之前设置,两分支都不再覆盖它。)

- [ ] **Step 5: 运行测试,确认通过 + 全仓编译**

Run: `go test ./pkg/store/xiaomi/ && go build ./...`
Expected: PASS + 无输出。

- [ ] **Step 6: 提交**

```bash
git add pkg/store/xiaomi/xiaomi.go pkg/store/xiaomi/xiaomi_test.go
git commit -m "feat(xiaomi): audit 新增上下架推断(在架/未上架)"
```

---

## Task 5: 应用宝 —— listing 未上架推断 + approved_first

**Files:**
- Modify: `pkg/store/tencent/tencent.go`(`audit()` 约 `tencent.go:371-402`)
- Test: `pkg/store/tencent/tencent_test.go`

**Interfaces:**
- Consumes: Task 1 的 `store.ListingState`、`store.AuditApprovedFirst`。
- Produces: `tencentListing(liveVersion string) store.ListingState`;`applyTencentFirstListing(state store.AuditState, liveVersion string) store.AuditState`。

应用宝审核态来自 `/query_app_update_status`;上下架只能靠「公开详情页是否抓到在架版本」推断 on_shelf/not_listed,**无法识别 off_shelf**(spec §7.5)。approved_first:审核通过(state==approved)且公开页无在架版本(liveVersion=="")。

- [ ] **Step 1: 写失败测试(扩展 tencent_test.go)**

在 `pkg/store/tencent/tencent_test.go` 追加:

```go
func TestTencentListing(t *testing.T) {
	if got := tencentListing("1.2.0"); got != store.ListingOnShelf {
		t.Errorf("tencentListing(non-empty) = %q, want on_shelf", got)
	}
	if got := tencentListing(""); got != store.ListingNotListed {
		t.Errorf("tencentListing(empty) = %q, want not_listed", got)
	}
}

func TestApplyTencentFirstListing(t *testing.T) {
	// 通过 + 无在架版本 → 首次上架通过
	if got := applyTencentFirstListing(store.AuditApproved, ""); got != store.AuditApprovedFirst {
		t.Errorf("apply(approved, empty) = %q, want approved_first", got)
	}
	// 通过 + 有在架版本 → 保持 approved(更新通过)
	if got := applyTencentFirstListing(store.AuditApproved, "1.2.0"); got != store.AuditApproved {
		t.Errorf("apply(approved, live) = %q, want approved", got)
	}
	// 非通过态不受影响
	if got := applyTencentFirstListing(store.AuditReviewing, ""); got != store.AuditReviewing {
		t.Errorf("apply(reviewing, empty) = %q, want reviewing", got)
	}
}
```

需确认 `tencent_test.go` 顶部已 import `"github.com/KevinGong2013/apkgo/v3/pkg/store"`;若无则加上。

- [ ] **Step 2: 运行测试,确认失败**

Run: `go test ./pkg/store/tencent/ -run 'TestTencentListing|TestApplyTencentFirstListing'`
Expected: 编译错误 `undefined: tencentListing`。

- [ ] **Step 3: 加两个纯函数**

在 `pkg/store/tencent/tencent.go` 的 `audit()` 之后新增:

```go
// tencentListing 由公开详情页是否抓到在架版本推断上下架:抓到→on_shelf,
// 抓不到→not_listed。应用宝无上下架接口,无法识别"下架"(spec §7.5)。
func tencentListing(liveVersion string) store.ListingState {
	if liveVersion != "" {
		return store.ListingOnShelf
	}
	return store.ListingNotListed
}

// applyTencentFirstListing 把"审核通过且无在架版本"细化为首次上架通过。
func applyTencentFirstListing(state store.AuditState, liveVersion string) store.AuditState {
	if state == store.AuditApproved && liveVersion == "" {
		return store.AuditApprovedFirst
	}
	return state
}
```

- [ ] **Step 4: wire 进 audit()**

在 `pkg/store/tencent/tencent.go` 的 `audit()` 里,把抓取公开页那段:

```go
	if live, ok := liveVersionFromStorePage(ctx, pkg); ok {
		res.LiveVersionName = live
	}
	return res
```

改为(抓取后统一设置 listing,并叠加 approved_first):

```go
	if live, ok := liveVersionFromStorePage(ctx, pkg); ok {
		res.LiveVersionName = live
	}
	res.Listing = tencentListing(res.LiveVersionName)
	res.State = applyTencentFirstListing(res.State, res.LiveVersionName)
	return res
```

- [ ] **Step 5: 运行测试,确认通过**

Run: `go test ./pkg/store/tencent/`
Expected: PASS(既有 `findVersionName` 等测试不受影响)。

- [ ] **Step 6: 提交**

```bash
git add pkg/store/tencent/tencent.go pkg/store/tencent/tencent_test.go
git commit -m "feat(tencent): audit 新增上下架推断与首次上架识别"
```

---

## Task 6: 荣耀 —— listing 恒 unknown(无接口)

**Files:**
- Modify: `pkg/store/honor/honor.go`(`audit()` 约 `honor.go:60-81`)

**Interfaces:**
- Consumes: Task 1 的 `store.ListingUnknown`。

荣耀无可靠上下架接口(`get-app-current-release.releaseStatus` 未使用且不可靠,spec §7.6)。listing 恒 `ListingUnknown`,如实表达"该店无此能力"。review 维度保持现有(需 releaseId);不产 approved_first —— 那需要在 auditByRelease 里额外调 get-app-detail,违反"零新增 API 调用",本次不做。

- [ ] **Step 1: 在 audit() 里设置 listing**

在 `pkg/store/honor/honor.go` 的 `audit()` 里,`appID` 解析成功之后、`if q.ExternalID != ""` 分支之前,加一行:

```go
	res.Listing = store.ListingUnknown // 荣耀无上下架查询接口
```

(auditByRelease / auditLiveVersionOnly 都不覆盖 res.Listing,该值保留。)

- [ ] **Step 2: 确认编译 + 既有测试全绿**

Run: `go test ./pkg/store/honor/ && go build ./...`
Expected: PASS + 无输出(既有 honor 审核测试不受影响)。

- [ ] **Step 3: 提交**

```bash
git add pkg/store/honor/honor.go
git commit -m "feat(honor): audit listing 维度如实报 unknown(无上下架接口)"
```

---

## Task 7: vivo —— online-state 解码(需核实字段)+ 安全降级

**Files:**
- Modify: `pkg/store/vivo/vivo.go`(`appDetails` 约 `vivo.go:540-546`、`audit()` 约 `:65`)
- Test(新建): `pkg/store/vivo/vivo_test.go`

**Interfaces:**
- Consumes: Task 1 的 `store.ListingState`。
- Produces: `vivoListing(onlineState int) store.ListingState`。

**关键前提**:`app.query.details` 返回体注释(`vivo.go:537-539`)称含 "online state",但**字段名与枚举值现有代码没有**。本任务先落"安全默认 = unknown"(字段缺失/未确认时不产出错误数据),再在字段确认后启用解码。**降级是本任务的验收底线**:即便字段名/枚举核不到,vivo listing 必须是 `ListingUnknown`,绝不产出错误的在架/下架。

- [ ] **Step 1: 写测试(vivoListing 纯函数,基于待确认枚举的假设值)**

创建 `pkg/store/vivo/vivo_test.go`。注意:枚举取值是**假设**,Step 3 用官方文档确认后再回填真实值;测试同步更新。默认分支(未知码值→unknown)是安全底线,必须存在:

```go
package vivo

import (
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

func TestVivoListing(t *testing.T) {
	// 假设枚举(Step 3 用 vivo 官方文档确认后回填真实码值):
	//   1 = 在架, 2 = 下架。未知码值一律 unknown(安全底线)。
	cases := map[int]store.ListingState{
		1: store.ListingOnShelf,
		2: store.ListingOffShelf,
		0: store.ListingUnknown, // 字段缺失/未返回 → 安全降级
		9: store.ListingUnknown, // 未知码值 → 安全降级
	}
	for code, want := range cases {
		if got := vivoListing(code); got != want {
			t.Errorf("vivoListing(%d) = %q, want %q", code, got, want)
		}
	}
}
```

- [ ] **Step 2: 运行测试,确认失败**

Run: `go test ./pkg/store/vivo/ -run TestVivoListing`
Expected: 编译错误 `undefined: vivoListing`。

- [ ] **Step 3: 核实 vivo online-state 字段名与枚举(实现阶段验证门)**

用以下手段之一确认 `app.query.details` 响应里"在架状态"的**确切 JSON 字段名**与**枚举取值**:
1. vivo 开放平台官方文档(https://dev.vivo.com.cn/ 开发者服务 → 应用发布 API,查 `app.query.details` 响应字段表);
2. 用真实 vivo 账号抓一次 `app.query.details` 原始响应,观察在架应用返回的额外字段;
3. 若都拿不到 —— **走安全降级**:字段名保持不接线(不新增 struct 字段),`audit()` 里 vivo listing 恒 `ListingUnknown`,Step 4/5 的字段解码跳过,只保留 `vivoListing` 纯函数 + 其 default→unknown 测试。并在提交信息与 spec §10.1 注明"待核实"。

把确认到的「字段名 + 码值→ListingState」写下来,回填到 Step 1 测试与 Step 4 代码。

- [ ] **Step 4: 加 vivoListing;字段确认后接线**

在 `pkg/store/vivo/vivo.go` 的 `mapVivoAuditState` 之后新增(码值以 Step 3 确认结果为准,下方为假设占位,确认后替换 case 值):

```go
// vivoListing 把 app.query.details 的在架状态码映射到上下架维度。
// 码值以 vivo 官方文档为准(见 plan Task 7 Step 3);未知/缺失一律 unknown。
func vivoListing(onlineState int) store.ListingState {
	switch onlineState {
	case 1: // 在架(以官方文档确认为准)
		return store.ListingOnShelf
	case 2: // 下架(以官方文档确认为准)
		return store.ListingOffShelf
	default:
		return store.ListingUnknown
	}
}
```

**仅当 Step 3 确认了字段名**,给 `appDetails` 结构体加字段(JSON tag 用确认的字段名,下方 `onlineStatus` 为假设):

```go
	OnlineStatus lenientInt `json:"onlineStatus"` // 在架状态,字段名以 vivo 文档确认为准
```

并在 `audit()` 里 `res.State, res.Detail = mapVivoAuditState(...)` 之后加:

```go
	res.Listing = vivoListing(int(app.OnlineStatus))
```

**若 Step 3 未能确认**:不加 struct 字段,改为在 `audit()` 里直接:

```go
	res.Listing = store.ListingUnknown // vivo 在架字段待核实(spec §10.1)
```

- [ ] **Step 5: 运行测试,确认通过 + 全仓编译**

Run: `go test ./pkg/store/vivo/ && go build ./...`
Expected: PASS + 无输出。

- [ ] **Step 6: 提交**

```bash
git add pkg/store/vivo/vivo.go pkg/store/vivo/vivo_test.go
git commit -m "feat(vivo): audit 新增上下架维度(online-state 解码/安全降级)"
```

---

## Task 8: 渲染层 + JSON + 文档

**Files:**
- Modify: `cmd/audit.go`(`auditOneLiner` 约 `:141-154`、`auditGlyph` 约 `:157-170`)
- Modify: `CLAUDE.md`(audit 段落)

**Interfaces:**
- Consumes: `apkgo.AuditStoreResult.Listing`(经 Task 1 内嵌自动获得);`store.ListingState` 各值。
- Produces: `listingGlyph(state string) (icon, color string)`;`auditOneLiner` 输出前置一列上下架状态。

JSON 输出无需改动:`AuditStoreResult` 内嵌 `store.AuditResult`,新增的 `listing` 字段自动出现在 `writeOutput` 的 JSON 里。本任务只做 text 渲染 + 文档。

- [ ] **Step 1: 加 listingGlyph,并给 auditGlyph 补两个新值图标**

在 `cmd/audit.go` 的 `auditGlyph` 里,`case "reviewing"` 之后新增:

```go
	case "approved_first":
		return "🎉", "32"
	case "needs_fix":
		return "🛠", "33"
```

在 `auditGlyph` 之后新增:

```go
// listingGlyph maps a ListingState string to an icon + ANSI colour for the
// text renderer's leading 上下架 column.
func listingGlyph(state string) (icon, color string) {
	switch state {
	case "on_shelf":
		return "🟢在架", "32"
	case "off_shelf":
		return "🔴下架", "31"
	case "not_listed":
		return "⚪未上架", ""
	case "unknown":
		return "❔未知", ""
	default:
		return "", ""
	}
}
```

- [ ] **Step 2: 在 auditOneLiner 前置上下架列**

把 `cmd/audit.go` 的 `auditOneLiner` 尾部:

```go
	icon, code := auditGlyph(string(r.State))
	line := icon + " " + string(r.State)
	if r.Detail != "" {
		line += " (" + r.Detail + ")"
	}
	return colorize(code, tty, line)
```

改为:

```go
	icon, code := auditGlyph(string(r.State))
	line := icon + " " + string(r.State)
	if r.Detail != "" {
		line += " (" + r.Detail + ")"
	}
	review := colorize(code, tty, line)
	if r.Listing != "" {
		licon, lcode := listingGlyph(string(r.Listing))
		if licon != "" {
			return colorize(lcode, tty, licon) + "  " + review
		}
	}
	return review
```

- [ ] **Step 3: 手动验证 text 渲染 + JSON 字段(编译 + help)**

Run: `go build ./... && go vet ./cmd/ ./pkg/store/...`
Expected: 无输出(编译 + vet 通过)。

手动核对(无凭据环境下用一个不存在的包名,确认命令不 panic、JSON 结构含 listing 字段路径正确):

Run: `go run . audit -p com.example.nonexistent -s huawei -o json 2>/dev/null || true`
Expected: 输出结构化 JSON(各店 error 字段有值即可),不 panic。

- [ ] **Step 4: 更新 CLAUDE.md 的 audit 段落**

在 `CLAUDE.md` 的「Review status (`apkgo audit`)」段落末尾补一段,说明新增维度(措辞与既有文档风格一致):

```markdown
`apkgo audit` 现同时报告两个正交维度:**上下架状态** `listing`
(`on_shelf` 在架 / `off_shelf` 下架 / `not_listed` 未上架 / `unknown`)与
**审核状态** `state`(原有 reviewing/approved/rejected/withdrawn/unknown,
新增 `approved_first` 首次上架通过、`needs_fix` 待整改)。复合态由两维度组合
表达(如"下架整改" = `listing:off_shelf` + `state:needs_fix`)。各店精度不同:
华为可精确给出 listing;OPPO 关键词推断;小米/应用宝只能判在架/未上架(识别
不了下架);荣耀无上下架接口(报 unknown);vivo 视字段可用性而定。
```

- [ ] **Step 5: 全仓测试 + 提交**

Run: `go test ./...`
Expected: PASS(全仓)。

```bash
git add cmd/audit.go CLAUDE.md
git commit -m "feat(audit): text 渲染新增上下架列并更新文档"
```

---

## Self-Review(计划自审记录)

- **Spec 覆盖**:§5 类型→Task 1;§7.1 华为→Task 2;§7.2 OPPO→Task 3;§7.4 小米→Task 4;§7.5 应用宝→Task 5;§7.6 荣耀→Task 6;§7.3 vivo→Task 7;§8 命令/输出→Task 8;§11 测试→各 Task 内 TDD;§10 开放项→Task 2(华为 case 9 / releaseState=3 已在映射内定型)、Task 7(vivo 字段核实门)、Task 4/5(小米/应用宝识别不了下架已注明)。§12 文件清单与本计划文件结构一致。
- **占位符扫描**:仅 Task 7 的 vivo 枚举码值是"待官方文档确认"——已显式设为验证门(Step 3),且安全降级(未知→unknown)有真实代码与测试兜底,不产出错误数据。其余步骤均含完整可运行代码。
- **类型一致**:`ListingState`/`Listing` 字段/`AuditApprovedFirst`/`AuditNeedsFix` 在 Task 1 定义,后续 Task 全部按此签名消费;各纯函数名(`mapHuaweiListing`/`reviewFromReleaseState`/`oppoListing`/`xiaomiListing`/`tencentListing`/`applyTencentFirstListing`/`vivoListing`/`listingGlyph`)前后引用一致。

---

## 交付顺序

Task 1(地基)→ Task 2(华为,验证整套模型)→ Task 3–6(OPPO/小米/应用宝/荣耀,无外部依赖)→ Task 7(vivo,含文档核实门)→ Task 8(渲染 + 文档)。Task 2–7 相互独立,可并行由不同 subagent 执行;均依赖 Task 1 先落地。
