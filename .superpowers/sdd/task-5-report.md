# Task 5 报告

## 目标
为腾讯应用宝的 `audit()` 增加 listing 未上架推断，并在“审核通过 + 无在架版本”时细化为 `approved_first`。

## 实现
- 新增 `tencentListing(liveVersion string) store.ListingState`
- 新增 `applyTencentFirstListing(state store.AuditState, liveVersion string) store.AuditState`
- 在 `audit()` 中接入 `res.Listing` 和 `res.State` 的最终赋值
- 在 `tencent_test.go` 增加两个纯函数测试

## 验证
- `GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/tencent/ -run 'TestTencentListing|TestApplyTencentFirstListing'`
- `GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/tencent/`

## 备注
- 未处理仓库里已存在的无关基线失败：`pkg/apkgo TestDiagnose_RealProbe`
