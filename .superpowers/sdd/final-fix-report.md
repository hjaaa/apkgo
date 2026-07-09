# 最终审查修复报告

## 修改概览

1. 腾讯 audit 公开页 enrichment 改为三态：
   - `on_shelf`: 成功抓到 live version
   - `not_listed`: 成功解析到目标包，但 live version 为空
   - `unknown`: 非 200、网络失败、页面结构异常、无法确认目标包
2. `approved_first` 仅在 `state=approved` 且 `listing=not_listed` 时产生；公开页失败不再伪造 `not_listed`，也不再把 `approved` 改写成 `approved_first`。
3. 补充腾讯非 200 / 网络失败测试，补充 `cmd/audit.go` 渲染回归测试。
4. 同步 `cmd/audit.go` help 与 `CLAUDE.md` 的状态集合说明。
5. 在 `pkg/store/huawei/audit_test.go` 的 `TestMapHuaweiReleaseState` 补上 case `9 -> needs_fix`。

## TDD 记录

先加测试，再跑到红：

```bash
GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/tencent/
```

结果：失败，原因符合预期，测试先要求腾讯 listing / first-listing 逻辑支持三态，而生产代码仍是旧的二态签名与语义。

新增的 `cmd` 与华为回归测试在加入时已是现有行为覆盖：

```bash
GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./cmd/
GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/huawei/
```

结果：通过。

## 验证命令与结果

```bash
gofmt -w pkg/store/tencent/tencent.go pkg/store/tencent/tencent_test.go cmd/audit.go cmd/audit_test.go pkg/store/huawei/audit_test.go
```

结果：通过。

```bash
GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/tencent/
```

结果：通过。

```bash
GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./cmd/
```

结果：通过。

```bash
GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./pkg/store/huawei/
```

结果：通过。

```bash
GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go build ./...
```

结果：通过。

```bash
GOCACHE=/private/tmp/apkgo-gocache GOMODCACHE=/private/tmp/apkgo-gomodcache go test ./...
```

结果：存在已知基线失败，未修改：

- `pkg/apkgo TestDiagnose_RealProbe`
  - 失败信息：`expected AnyFailed=true with bogus api_key`

## 备注

- 未触碰未跟踪的 `.cache/` 与根 `AGENTS.md`。
- 仅修改允许文件列表中的文件。
