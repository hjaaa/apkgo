# Task 3 报告：OPPO listing 关键词 + needs_fix

## 完成内容
- 在 `pkg/store/oppo/oppo.go` 增加 `oppoListing(name string) store.ListingState`，基于 `audit_status_name` 关键词推导上下架状态。
- 在 `mapOppoAudit` 中加入“整改”→ `store.AuditNeedsFix` 分支，优先级放在拒绝前。
- 在 `audit()` 中将 `res.Listing` 接到 `oppoListing(app.AuditStatusName)`。

## 测试
- 新增 `pkg/store/oppo/oppo_test.go`，覆盖 listing 映射与 `待整改` 分支。
- 验证命令：
  - `GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/oppo/ -run 'TestOppoListing|TestMapOppoAuditNeedsFix'`
  - `GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/oppo/`

## 已知保留项
- 未处理仓库中既有的全量基线失败：`pkg/apkgo TestDiagnose_RealProbe`。
