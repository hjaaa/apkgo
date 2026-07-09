# Task 2 报告：华为 listing 精确映射 + approved_first + needs_fix(case 9)

## 实现内容
- 在 `pkg/store/huawei/audit_test.go` 增加 `TestMapHuaweiListing`，锁定华为 `releaseState + onShelfVersionCode` 到 `store.ListingState` 的精确映射矩阵。
- 在 `pkg/store/huawei/audit_test.go` 增加 `TestReviewFromReleaseState`，锁定 `approved_first` 与 `needs_fix(case 9)` 的审核态细化。
- 在 `pkg/store/huawei/huawei.go` 为 `mapHuaweiReleaseState` 增加 `case 9 -> store.AuditNeedsFix`。
- 在 `pkg/store/huawei/huawei.go` 新增 `reviewFromReleaseState` 与 `mapHuaweiListing`，并在 `audit()` 中接入：
  - `State/Detail` 由 `reviewFromReleaseState(...)` 生成
  - `Listing` 由 `mapHuaweiListing(...)` 生成

## 测试结果
- RED:
  - 命令：`GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/huawei/ -run 'TestMapHuaweiListing|TestReviewFromReleaseState'`
  - 结果：失败
  - 证据：`undefined: mapHuaweiListing`、`undefined: reviewFromReleaseState`
- GREEN:
  - 命令：`GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/huawei/`
  - 结果：`ok github.com/KevinGong2013/apkgo/v3/pkg/store/huawei 0.642s`

## TDD RED/GREEN 证据
1. 先追加测试，再运行定向测试，确认因缺少待实现函数而 RED。
2. 再做最小生产代码实现与 `audit()` 接线。
3. 最后运行华为包全量测试，确认 GREEN。

## 文件变更
- `pkg/store/huawei/audit_test.go`
- `pkg/store/huawei/huawei.go`
- `.superpowers/sdd/task-2-report.md`

## 自审
- 改动范围保持在任务要求的华为实现与测试内，未触碰无关基线失败。
- `approved_first` 只在原始审核态为 `approved` 且 `onShelfVersionCode == 0` 时触发，没有扩大判断面。
- `listing` 作为独立维度接入 `audit()`，不影响现有版本号字段填充。
- `case 9` 从 `unknown` 收敛为 `needs_fix`，与 brief 要求一致。

## concerns
- 未运行仓库 `go test ./...`，因为任务说明已明确存在无关基线失败：`pkg/apkgo TestDiagnose_RealProbe`。
