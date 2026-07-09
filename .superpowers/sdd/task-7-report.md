# Task 7 报告：vivo online-state 安全降级

## 结论

本任务按 brief 的安全降级路径完成，没有新增或接线任何未核实的 `app.query.details` online-state 字段。

## 原因

主控已做过一次公开搜索，未找到可可靠引用的 vivo 官方 `app.query.details` 上下架字段名和枚举值。本任务遵循约束：

- 不猜测字段名；
- 不猜测 `1=在架`、`2=下架` 等枚举；
- `audit()` 明确返回 `store.ListingUnknown`；
- 保留 `vivoListing(onlineState int)` 纯函数，但当前对所有输入都返回 `store.ListingUnknown`，避免产出伪造的上架/下架结论。

## TDD 过程

1. 先新增 `TestVivoListingFallsBackToUnknownWhenOnlineStateIsUnverified`。
2. 运行 RED：
   - `go test ./pkg/store/vivo/ -run TestVivoListingFallsBackToUnknownWhenOnlineStateIsUnverified`
   - 结果按预期失败：`undefined: vivoListing`
3. 最小实现：
   - 新增 `vivoListing`，恒定返回 `store.ListingUnknown`
   - 在 `audit()` 中设置 `res.Listing = store.ListingUnknown`
4. 运行验证：
   - `go test ./pkg/store/vivo/`
   - `go build ./...`

## 改动范围

- `pkg/store/vivo/vivo.go`
- `pkg/store/vivo/vivo_test.go`

## 后续

若后续拿到 vivo 官方文档或真实 `app.query.details` 响应样本，再把字段名和枚举值回填到 `appDetails` 与 `vivoListing`，并同步更新测试。
