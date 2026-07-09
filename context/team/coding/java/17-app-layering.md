# 应用分层

各 `ai-work-*` 服务内部，**核心业务请求链**统一为 **controller → service（接口 + ServiceImpl，MyBatis-Plus IService）→ mapper（BaseMapper + XML）** 三层；不设 Manager 层，通用能力下沉到 `ai-work-common` 对应模块；跨服务调用走 Feign / 网关。三层约束仅针对业务请求链：config（配置）、handler（全局处理器）、feign（跨服务客户端）、security（安全组件）等支撑包按职责独立存在，不强行归入三层，也不要把策略、适配器、第三方封装硬塞进 ServiceImpl。

1. 【推荐】分层依赖必须单向：controller 依赖 service，service 依赖 mapper，禁止反向依赖与跨层调用（如 controller 直接调用 mapper）。对第三方平台的封装、多 mapper 的组合复用等通用逻辑收敛在 service 层，可跨服务复用的抽取到 `ai-work-common`，不散落在 controller。

2. 【参考】（分层异常处理规约）mapper 层产生的异常无需捕获打印，向上抛出由 service 层统一处理；service 层出现异常时，必须记录出错日志到磁盘，尽可能带上参数和上下文信息，相当于保护案发现场；controller 层不继续向上抛异常，由全局异常处理器（`GlobalBizExceptionHandler`）统一转为 `R` 错误响应返回。

3. 【参考】分层领域模型规约：

    - DO（Data Object）：此对象与数据库表结构一一对应，通过 mapper 层向上传输数据源对象。
    - DTO（Data Transfer Object）：数据传输对象，service 向外传输的对象。
    - BO（Business Object）：业务对象，可以由 service 层输出的封装业务逻辑的对象。
    - Query：数据查询对象，各层接收上层的查询请求。注意超过 2 个参数的查询封装，禁止使用 Map 类来传输。
    - VO（View Object）：显示层对象，通常是返回给前端渲染的对象。
