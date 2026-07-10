# 各渠道应用商店 API 状态能力与文档链接总表

> 调研日期：2026-07-10。基于各平台官方 OpenAPI 文档逐一核实（SPA 站点均经浏览器渲染
> 或官方同源数据接口验证原文）。覆盖维度：上架状态（未上架/在架/已下架）、审核状态
> （审核中/驳回/通过/通过-首次上架/待整改）、驳回/整改理由、下架理由。
>
> 符号约定：✅=官方字段直接可实现　⚠️=需推断/有条件/无契约　❌=不可实现

## 能力对照表

| 渠道 | 上架状态（未上架/在架/已下架） | 审核状态（审核中/驳回/通过/通过-首次上架/待整改） | 驳回/整改理由 | 下架理由 |
|---|---|---|---|---|
| **华为** | ✅✅✅ 三档全可实现（`releaseState` 14 值码表精确区分） | ✅✅✅／⚠️推断（过审+`onShelfVersion*`为空，可靠）／❌无整改状态 | ✅ `auditInfo.auditOpinion`；大陆应用另有版权/版号/备案三组 `*AuditOpinion` | ❌ 仅 AGC 控制台/站内信 |
| **荣耀** | ⚠️⚠️❌ 未上架/在架可弱推断（`releaseInfo` 空/非空）；已下架不可实现 | ✅✅✅（`auditResult` 0/1/2）／⚠️推断／❌ | ✅ `auditMessage` + `auditAttachment` | ❌ 无下架状态字段 |
| **OPPO** | ⚠️✅✅ 在架/已下架可实现（`state` 1/2 或 `audit_status` 111/222）；未上架需推断（`audit_status`=0） | ✅✅✅（17 值码表）／⚠️推断／⚠️后台文本标签兜底 | ✅ 最全：`refuse_reason`、`refuse_advice`、`refuse_file`、`business_refuse_reason`、`freeze_reason/advice` | ⚠️ 仅 `offline_info`（开发者自行申请下架）；平台强制下架无字段 |
| **vivo** | ✅✅✅ 三档全可实现（`saleStatus` 0/1/2 一一对应） | ✅✅✅（`status` 2/4/3）／⚠️推断（status=3+saleStatus=0）／❌ | ✅ `unPassReason`（仅审核驳回场景） | ❌ 官方明文：通过消息系统/邮箱通知 |
| **腾讯** | ⚠️⚠️❌ 未上架可推断（错误码 1000011）；在架靠爬公开页（无契约）；已下架不可区分 | ✅✅✅（`audit_status` 1/2/3，8=撤回）／⚠️推断（意义有限）／❌ | ✅ `audit_reason` | ❌ 下架是人工工单流程 |
| **三星** | ⚠️✅✅ 在架（`FOR_SALE`）/已下架（`SUSPENDED`/`TERMINATED`）可实现；未上架需 3100 探测法推断 | ✅✅✅（`contentStatus` ~40 值）／⚠️推断（3100 探测法）／⚠️推断（`*_SUSPENDED`） | ❌ 无字段（`reviewComment` 是开发者→审核方，方向相反） | ❌ 仅 Seller Portal/邮件 |
| **小米** | ⚠️✅❌ 在架可实现（`onlineVersionCode`>0）；未上架与已下架混同（同为 0） | ⚠️❌⚠️（版本比对推断，审核中与驳回无法区分）／⚠️需历史状态／❌ | ❌ 无任何接口 | ❌ 无任何接口 |

> 审核状态列的三个符号依次对应：审核中·驳回·通过／通过-首次上架／待整改

## API 文档链接表

| 渠道 | 关键接口 | 官方文档链接 | 访问说明 |
|---|---|---|---|
| 华为 | `GET /api/publish/v2/app-info`（releaseState、onShelfVersion*、auditInfo） | [查询应用信息 v2](https://developer.huawei.com/consumer/cn/doc/AppGallery-connect-References/agcapi-app-info-query-0000001158365045) ・ [v3 (HarmonyOS)](https://developer.huawei.com/consumer/cn/doc/App/agc-help-publish-api-appinfo-query-0000002236041422) | SPA，需浏览器打开 |
| 荣耀 | `get-audit-result` / `get-app-current-release` / `get-app-detail`（auditResult、releaseInfo） | [API传包服务指引（13 个接口同页）](https://developer.honor.com/cn/doc/guides/101359) | SPA，需浏览器渲染，无需登录 |
| OPPO | `GET /resource/v1/app/info`（audit_status、state、refuse_*） | 官方：[open.oppomobile.com](https://open.oppomobile.com/new/developmentDoc/info?id=12971)（需登录）；镜像：[查询普通包详情](https://www.yimenapp.com/kb-yimen/10842/) ・ [审核状态对照表](https://www.yimenapp.com/kb-yimen/10834/) | 官方站需登录，镜像经开源实现交叉验证 |
| vivo | `app.query.details`（status、saleStatus、unPassReason） | [查询详细信息 doc/346](https://dev.vivo.com.cn/documentCenter/doc/346) ・ [参数字典 doc/344](https://dev.vivo.com.cn/documentCenter/doc/344) ・ [违规处罚规则 doc/479](https://dev.vivo.com.cn/documentCenter/doc/479) | SPA；原文可从 `dev.vivo.com.cn/webapi/doc/info?id=<docId>` 直接验证 |
| 腾讯 | `/query_app_update_status`（audit_status、audit_reason） | [API更新应用信息](https://wikinew.open.qq.com/index.html#/iwiki/4015262492) ・ [应用下线申请（人工工单）](https://wikinew.open.qq.com/index.html#/iwiki/4007776090) | SPA；原文可从 `material.open.qq.com/openwiki/getOpenWikiHtml?directoryId=4015262492` 验证 |
| 三星 | `GET /seller/contentInfo`（contentStatus）、`contentStatusUpdate` | [contentInfo](https://developer.samsung.com/galaxy-store/galaxy-store-developer-api/content-publish-api/view-sellers-app-details.html) ・ [contentStatus 映射表](https://developer.samsung.com/galaxy-store/galaxy-store-developer-api/content-publish-api/status-parameters-mapping.html) ・ [上下架流转](https://developer.samsung.com/galaxy-store/galaxy-store-developer-api/content-publish-api/change-app-status.html) | 普通 HTML，直接可访问 |
| 小米 | `/dev/query`（onlineVersionCode；无审核接口） | [应用自动发布接口 pId=1134（含"暂未开放"FAQ）](https://dev.mi.com/xiaomihyperos/documentation/detail?pId=1134) ・ [自更新技术指引 pId=2007（onlineVersionCode 定义）](https://dev.mi.com/xiaomihyperos/documentation/detail?pId=2007) | SPA，需浏览器渲染 |

## 补充说明

- **首次上架（approved_first）**：所有平台均无官方字段，只能组合推断。华为依据最可靠
  （`onShelfVersion*` 为空是官方定义的"无在架版本"信号）；三星可用官方 3100 探测法
  （按 `appStatus=SALE` 查 `stagedRolloutBinary` 返回错误码 3100 即无在架版本）。
- **待整改（needs_fix）**：无任何平台的官方码表包含"整改"状态。OPPO 后台实际返回过
  含"整改"的 `audit_status_name` 文本标签（本仓库测试用例有实测样本），只能关键词兜底。
- **下架理由**：全平台一致不通过 API 提供。vivo 官方原文（doc/479）："下架原因可能会
  通过消息系统、开发者在vivo开放平台注册时填写的邮箱或其他合理的方式进行通知"。
- **腾讯 audit_status=0**：官方码表只有 1/2/3/8 四个值，0（"no submission"）是实测
  观察行为，非官方定义。
