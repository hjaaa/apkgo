# apkgo audit 能力缺口补齐改造清单（设计）

> 日期：2026-07-10。
> 输入：`docs/store-api-capability.md`（各渠道官方 API 能力调研）与 `pkg/store/` 各家
> audit 实现的逐项差异对照。
> 结论形态：按优先级 P0–P4 编排的改造清单，先经团队评审，评审通过后在本仓库落地。

## 1. 背景与差异摘要

`docs/store-api-capability.md` 描述的是**官方 API 能力上限**，代码是**实际落地情况**。
逐店对照后差异分三类：

1. **口径冲突**（文档与代码必有一方错）：华为 `releaseState=9` 被映射为
   `needs_fix`（huawei.go:127），文档说华为无整改态；三星 `SUSPENDED` 被映射为
   `approved`（samsung.go:170），语义更接近已下架/待整改。
2. **能力未落地**（文档✅、代码没做）：华为驳回理由、vivo listing 与驳回理由、
   三星 listing、OPPO 4 组理由字段与 not_listed、荣耀 listing 弱推断与附件、
   荣耀/vivo/三星 approved_first。
3. **实现路径不同但行为等价**（不改代码，只需文档补注）：OPPO 靠
   `audit_status_name` 关键词而非数字码、三星靠关键词而非全码表、小米判据用
   `packageInfo` 存在性而非 `onlineVersionCode`。

## 2. 目标与约束

- **目标**：补齐审核状态、上架状态（listing）、驳回/整改理由三类缺口，并修正语义
  错误的既有映射。
- **约束（已与需求方确认）**：
  - 无任何商店的真实凭证可实测。落地策略统一为：**按官方文档码表实现映射；字段
    缺失或出现意外值一律降级 `unknown`，绝不臆测；用官方文档样本固化表驱动单测**。
  - 不新增 `AuditState` / `ListingState` 枚举值，只在现有 7+4 个状态内工作。
  - 小步提交：每个优先级批次是独立可回滚的提交序列，批内一店一提交；
    重构（P0 映射修正）与新功能（P1–P3）分开提交。

## 3. 优先级编排

采用**维度横切**：P0 语义正确性 → P1 驳回/整改理由 → P2 listing →
P3 approved_first → P4 文档同步。理由：

- 正确性修复影响 `--watch` 终态判断，改动最小、收益最直接，先行落地；
- `approved_first` 的推断依赖"是否已有在架版本"信号，该信号由 P2 产出，
  所以 P3 必须排在 P2 之后；
- 同一批次内改动模式相同（同为字段解析或同为映射调整），评审聚焦。

## 4. P0 语义正确性修正（3 项，只改既有映射）

共同背景：`unknown`/`needs_fix` 是非终态（`--watch` 轮询到超时），
`approved`/`rejected`/`withdrawn`/`approved_first` 是终态。

| 项 | 现状 | 决策 | 说明 |
|---|---|---|---|
| P0.1 华为 `releaseState=9`（下架审核不通过） | → `needs_fix`（huawei.go:127） | 改 → `rejected` | 需求方决策：优先保住终态可用性，接受"被驳回的是下架申请而非版本审核"的语义近似；Detail 保留原始码 `releaseState=9`。 |
| P0.2 三星 `SUSPENDED` 及 `*_SUSPENDED` 系列值 | 顶层 `SUSPENDED` → `approved`（samsung.go:170），`*_SUSPENDED` 落入其他关键词规则 | 统一改 → `needs_fix` | 与调研文档 `*_SUSPENDED`→待整改 的口径对齐；"已下架"语义由 P2 新增的三星 listing（`off_shelf`）承担。Detail 保留原始 contentStatus。 |
| P0.3 OPPO 关键词「冻结」 | 与「下架/撤销」一起 → `withdrawn` | 「冻结」改 → `needs_fix`；「下架/撤销」维持 `withdrawn` | 冻结有 `freeze_reason/advice` 官方字段支撑整改语义（P1.3 接入）；`withdrawn` 的含义在 P4 文档中明确为"本次提交已不在审核流程中"。 |

P0.1 落地时附带一个**零行为变更的核对步骤**：按官方 14 值码表逐值复核
`mapHuaweiReleaseState` 与 `mapHuaweiListing`（含 listing 中 9→`off_shelf` 是否成立
——若 9 为"下架申请被拒"，应用实际仍在架）。复核结果若与现有映射冲突，单独提出
讨论后再改，不混入本批提交。

受影响测试：`pkg/store/huawei/audit_test.go`（9 的断言）、
`pkg/store/samsung/audit_test.go`（SUSPENDED 断言）、`pkg/store/oppo/oppo_test.go`。

## 5. P1 驳回/整改理由接入（4 项，只加字段解析进 Detail）

改动模式统一：响应结构体加字段 → 对应状态时拼入 `Detail` → 官方文档样本单测。
空字段一律跳过；不改任何状态映射。

| 项 | 商店 | 字段 | 注入时机 |
|---|---|---|---|
| P1.1 | 华为 | `auditInfo.auditOpinion`；大陆应用另取版权/版号/备案三组 `*AuditOpinion` | `rejected` 时拼入 Detail（当前只有 `releaseState=N`） |
| P1.2 | vivo | `unPassReason` | status=4（`rejected`）时作为 Detail（当前为空字符串） |
| P1.3 | OPPO | `refuse_advice`、`business_refuse_reason`、`refuse_file`（链接）；`freeze_reason`/`freeze_advice` | `rejected` 时在既有 `refuse_reason` 后拼接；冻结（P0.3 后为 `needs_fix`）时用 freeze 组 |
| P1.4 | 荣耀 | `auditAttachment` | `rejected` 时在既有 `auditMessage` 后追加 |

不动：腾讯（`audit_reason` 已实现）、三星/小米（官方无理由字段）。

## 6. P2 listing 补齐（4 项）

改动模式统一：官方码表映射 + 意外值/缺失降级 `unknown` + 文档样本单测。

- **P2.1 vivo**（落差最大项）：解析 `saleStatus`，按官方参数字典（doc/344）
  0/1/2 → `not_listed`/`on_shelf`/`off_shelf`，替换 vivo.go:66 硬编码的
  `ListingUnknown`。**关键防线**：字段缺失时 Go int 零值为 0，会把"没返回"误报成
  `not_listed`——`saleStatus` 必须用指针类型判存在性，缺失 → `unknown`。
- **P2.2 三星**：新增 listing 维度（当前 `audit()` 从不赋值 `res.Listing`）：
  `FOR_SALE` → `on_shelf`；`SUSPENDED`（含 `*_SUSPENDED` 系列）/`TERMINATED` → `off_shelf`；
  `REGISTERING` 等明确未发布态 → `not_listed`；其余 → `unknown`。
  3100 探测法不在本批（放 P3，与 approved_first 共用一次探测调用）。
- **P2.3 荣耀**：用 `get-app-detail` 已获取的 `releaseInfo` 做弱推断：调用成功且
  releaseInfo 非空 → `on_shelf`，为空 → `not_listed`；调用失败 → `unknown`。
  已下架恒无法识别（荣耀无此字段），口径同步进 P4 文档。
- **P2.4 OPPO**：让已解析但闲置的 `audit_status` 数字码（oppo.go:598）参与判定：
  0 → `not_listed`（补上从不输出 not_listed 的缺口）、111 → `on_shelf`、
  222 → `off_shelf`；数字码不可识别时回落到现有关键词匹配。
  风险标注：OPPO 官方文档需登录，映射依据为经开源实现交叉验证的镜像文档。

## 7. P3 approved_first 补齐（3 项）

复用华为/腾讯已验证的模式：**`approved` + "无在架版本"信号 → `approved_first`**。
信号均由 P2 产出或本批新增：

- **P3.1 荣耀**：`get-audit-result` 报 approved 且 releaseInfo 为空（P2.3 信号）。
- **P3.2 vivo**：status=3 且 `saleStatus=0`（P2.1 信号），与调研文档推断法一致。
- **P3.3 三星**：官方 3100 探测法——按 `appStatus=SALE` 查 `stagedRolloutBinary`，
  返回错误码 3100 即"无在架版本"；approved + 3100 → `approved_first`。
  P3 中唯一新增 API 调用的项；探测失败降级为普通 `approved`，不影响主结果。

## 8. P4 文档与口径同步（纯文档）

- `docs/store-api-capability.md`：能力表补一列"本项目实现现状"；补注已定口径——
  华为 9→`rejected` 的取舍、OPPO/`withdrawn` 的扩展含义（"本次提交已不在审核
  流程中"）、小米判据实际用 `packageInfo` 存在性（与 `onlineVersionCode` 语义等价）。
- 项目 `CLAUDE.md` audit 段落：vivo listing 三档、三星 listing、needs_fix 渠道范围
  （OPPO/三星）、approved_first 支持范围（华为/腾讯/荣耀/vivo/三星）。

## 9. 不做及原因

| 项 | 原因 |
|---|---|
| 腾讯 1000011 推断未上架 | 无契约的实测观察，无凭证复验；现有爬页方案已覆盖 not_listed/on_shelf 且失败降级 unknown |
| OPPO approved_first | 过审后 `audit_status` 不再是 0，缺少可靠的"无在架版本"信号 |
| OPPO `offline_info`（下架理由） | 仅覆盖开发者自请下架场景，价值有限，且 audit 输出模型无下架理由专用位置；后续按需再议 |
| 荣耀已下架识别 | 官方无字段，不可实现 |
| 荣耀无 releaseId 时的审核状态 | 现状（State 留空）是刻意规避 current-release 串任务缺陷的保守设计，保留 |
| 小米 rejected / approved_first / needs_fix | 版本比对推断原理上无法区分，官方无接口 |

## 10. 测试与验证

- 每个改造项：官方文档样本 → 表驱动单测正例；字段缺失/意外值 → `unknown` 负例。
- 每次提交 `go test ./...` 通过为门槛；无凭证，不做真实 API 集成测试。
- 各批次验收标准：
  - P0：三处映射断言更新且通过；华为码表复核记录无未决冲突。
  - P1：四家 Detail 在驳回/整改样本下非空且内容与样本一致。
  - P2：四家 listing 在样本下产出预期三态；缺失/意外值样本产出 `unknown`。
  - P3：三家在"approved+无在架信号"样本下产出 `approved_first`，信号缺失时回落 `approved`。
  - P4：两份文档与代码行为一致（人工核对）。

## 11. 交付节奏与提交规范

- 顺序执行 P0 → P1 → P2 → P3 → P4；P3 依赖 P2，不可提前。
- 每批次一个提交序列，批内一店一提交。约定式提交：P0 用 `fix(<store>)`，
  P1–P3 用 `feat(<store>)`，P4 用 `docs`；message 用简体中文。

## 12. 风险

- **官方文档与线上实际返回不符**：主要风险。降级 `unknown` 兜底后果是"少报"而非
  "误报"；误报风险集中在文档样本正确但线上语义漂移的场景，无凭证下只能靠用户反馈
  回收。
- **OPPO 依据镜像文档**（官方站需登录），已经开源实现交叉验证，仍标为中风险。
- **vivo `saleStatus`** 此前被上游刻意标注 "unverified; degrade safely"，
  指针判存在性 + 缺失降级是关键防线。
