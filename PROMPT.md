# 消毒供应中心器械追溯服务（cssd-instrument-trace-service）

## 一、项目概述

基于 Go 实现的消毒供应中心器械追溯 Web 项目，一款后端服务，完成器械包登记、清洗消毒灭菌批次管理、发放回收闭环、灭菌参数合格判定与追溯查询。

项目类型：**全栈 Web 应用**（Go 后端服务 + `go:embed` 内嵌前端页面）。

## 二、业务背景与领域规则

医院消毒供应中心（CSSD）负责手术器械包的清洗、消毒、灭菌、发放与回收。每个器械包有唯一条码，经历「回收→清洗→消毒→灭菌→发放→使用→回收」的循环。系统记录每个环节的操作人、设备、时间与关键参数（灭菌温度、时长、压力），灭菌参数不合格的批次必须拦截不得发放。器械包按有效期管理，过期包禁止发放。整个循环要可追溯。

关键领域规则（这些规则是后续埋 bug 验证跨文件改动的核心约束，必须真实实现）：

1. 器械包循环状态机：待回收(to_collect) → 已回收(collected) → 清洗中(washing) → 已清洗(washed) → 灭菌中(sterilizing) → 已灭菌(sterilized) → 已发放(issued) → 使用中(in_use) → 回到待回收；循环必须按序推进，跳步拒绝。
2. 灭菌参数判定：每批次记录灭菌温度/时长/压力，与设定下限比较，任一项不达标判定灭菌失败(sterilization_failed)，批次内器械包全部拦截、不得发放。
3. 发放校验：器械包发放前必须校验「已灭菌 + 未过期 + 灭菌批次参数合格」三项，任一不满足拒绝发放。
4. 有效期管理：灭菌完成后按包类型计算有效期（如 7 天/14 天），到期自动置为过期；过期包需重新清洗灭菌才能再次发放。
5. 发放回收闭环：发放记录绑定使用科室与手术间，回收时按条码闭环；未回收器械包超过 24 小时进入「丢失待查」清单。
6. 追溯查询：按器械包条码可拉出完整循环记录（每环节时间、操作人、设备、参数），按灭菌批次可查该批所有器械包去向。

## 三、核心实体（≥3 个，必须贯穿全栈）

每个实体必须贯穿「数据库/存储表 → domain model → repository → service → handler → 前端 API 层 → 前端页面/组件」全链路。

| 实体 | 关键字段 | 业务动作 |
|---|---|---|
| 器械包 InstrumentPack | id、条码、包类型、内含器械清单、当前环节、有效期 | 登记、循环流转 |
| 灭菌批次 SterilizationBatch | id、灭菌器id、温度、时长、压力、结果(合格/失败)、时间 | 参数记录、判定 |
| 环节记录 CycleRecord | id、器械包id、环节、操作人、设备id、时间、参数快照 | 留痕、追溯 |
| 发放记录 IssueRecord | id、器械包id、科室、手术间、发放人、时间 | 发放、回收 |
| 灭菌器 Sterilizer | id、名称、状态(可用/维护) | 维护 |

## 四、核心页面与 API

### 前端页面（≥4 个路由，至少 2 个页面共用同一个业务组件）

| 项目 | 说明 |
|---|---|
| / 工作台总览 | 各环节在途器械包计数 + 灭菌失败拦截 + 丢失待查 | InstrumentPack |
| /packs 器械包管理 | 器械包列表 + 条码登记 + 环节流转 | InstrumentPack |
| /sterilization 灭菌管理 | 批次登记 + 参数录入 + 合格判定 | SterilizationBatch |
| /issue 发放回收 | 发放登记 + 回收扫码 | IssueRecord |
| /trace 追溯查询 | 按条码/批次追溯 | CycleRecord |

### 后端 REST API（与页面一一对应，命中真实业务链路）

| 项目 | 说明 |
|---|---|
| POST /api/packs | 登记器械包 |
| POST /api/packs/{id}/cycle | 环节流转（校验状态机顺序） |
| POST /api/sterilizations | 创建灭菌批次（录入参数） |
| POST /api/sterilizations/{id}/complete | 完成灭菌（参数判定 + 器械包状态更新） |
| POST /api/packs/{id}/issue | 发放器械包（校验灭菌/有效期/参数） |
| POST /api/packs/{id}/collect | 回收器械包（闭环校验） |
| GET /api/packs/{id}/trace | 器械包循环追溯 |
| GET /api/sterilizations/{id}/packs | 批次器械包去向 |
| GET /api/lost | 丢失待查清单 |
| GET /api/healthz | 健康检查 |

## 五、横切关注点（≥2 个）

1. 操作审计日志：环节流转、灭菌判定、发放回收全部留痕；触达 handler → service → audit store。
2. 过期扫描定时任务：每小时扫描过期器械包并更新状态；触达 service → store → 器械包列表。
3. 全局错误处理与统一响应格式。

## 六、共享枚举/常量（≥2 组）

枚举/常量要求前后端各自定义且保持一致，README 中列出所有出现位置。

1. 器械包环节 PackStage：to_collect / collected / washing / washed / sterilizing / sterilized / issued / in_use / expired。
2. 灭菌结果 SterilizeResult：pass / fail。
3. 包类型 PackType：surgical / dressing / instrument / implant。

## 七、共享前端组件与 hooks（组件 ≥3 个、hooks ≥2 个）

### 共享组件（放 `web/components/`）

1. StageBadge：环节徽标，被工作台与器械包列表共用。
2. CycleTimeline：循环时间线，被追溯页与器械包详情共用。
3. IssueForm：发放登记表单，被发放页与器械包详情共用。

### 自定义 hooks（放 `web/hooks/`）

1. usePacks(filter)：器械包列表，被列表页与工作台共用。
2. useTrace(id)：循环追溯，被追溯页与详情共用。

## 八、后端中间件（≥2 个）

1. auditLogger：审计日志中间件。
2. errorHandler：统一错误/panic 处理中间件。
3. issueGuard：发放前置校验守卫中间件。

## 九、技术要求

- 语言：**Go 1.23**（go.mod 声明 `go 1.23`，module 路径 `example.com/cssd-instrument-trace-service`）
- 运行：`go run .` 默认监听 `8080`，支持 `PORT` 环境变量覆盖
- 存储：SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 关闭）或内置内存仓储 + JSON 文件持久化，二选一，必须可重复构建、无外部服务依赖
- 前端：纯原生 HTML/CSS/JS，`go:embed` 内嵌 `web/` 静态资源，禁止引入外部 CDN 依赖（离线可跑）
- 服务入口：`GET /healthz` 返回 200；页面 `GET /` 可访问
- 根目录必须包含 `runtime_smoke.json`：`mode: service` + `start: go run .` + `ready_url: /healthz`；`project_intro` 一句话简介必须包含项目类型（如「基于 Go 实现的XXX Web 项目，一款后端服务，完成……」）
- 根目录必须包含 `README.md`：项目说明、目录结构、运行与测试命令、环境变量说明
- 构建：`go build ./...` 与 `go test ./...` 必须全部通过（基线干净、无 bug）

## 十、文件结构强制清单（规模目标：≥2000 行 Go 功能代码、≥20 个 `.go` 文件）

```
backend/
├── go.mod
├── main.go
├── config/
│   └── config.go            # 灭菌参数下限、有效期规则、回收时限
├── domain/
│   ├── pack.go              # 器械包实体 + 循环状态机
│   ├── sterilization.go     # 灭菌批次 + 参数判定
│   ├── cycle.go             # 环节记录
│   ├── issue.go             # 发放回收
│   └── sterilizer.go        # 灭菌器
├── store/
│   ├── pack_store.go
│   ├── sterilization_store.go
│   ├── cycle_store.go
│   ├── issue_store.go
│   ├── sterilizer_store.go
│   └── audit_store.go
├── service/
│   ├── pack_service.go      # 登记 + 环节流转
│   ├── sterilization_service.go # 批次 + 参数判定
│   ├── issue_service.go     # 发放校验 + 回收闭环
│   ├── trace_service.go     # 追溯聚合
│   ├── expiry_sweeper.go    # 过期扫描
│   └── audit_service.go
├── httpapi/
│   ├── router.go
│   ├── pack_handler.go
│   ├── sterilization_handler.go
│   ├── issue_handler.go
│   ├── trace_handler.go
│   └── health_handler.go
├── middleware/
│   ├── audit.go
│   ├── error_handler.go
│   └── issue_guard.go
└── web/
    ├── index.html
    ├── app.js
    ├── style.css
    ├── components/
    └── hooks/
```

**严禁合并职责到单一文件**：handler、service、repository、domain 必须分层；禁止把所有逻辑塞进 `main.go` 或一个 `handlers.go`。目标规模下限 2000 行 / 20 个 `.go` 文件，实际建议做到 3000 行以上 / 30 个文件以上，保证每个业务模块（实体、状态机、联动、报表）都有独立文件。

## 十一、运行、测试与交付要求

1. `go build ./...` 通过；`go test ./...` 全绿（含各业务模块的单元测试，测试文件不计入规模）。
2. `go run .` 后 `GET /healthz` 返回 200，前端页面 `GET /` 可打开且核心接口可用。
3. 每个核心业务动作都要有可复现的输入（API 请求/页面操作），方便后续构造缺陷与验证命令。
4. 代码中不得出现任何「故意埋错」「TODO bug」类注释；交付为干净基线。

## 十二、质量红线

1. **天然多文件、多层耦合**：任何一个小改动（如给某状态新增一个合法迁移）都应触达 3-5 个文件（domain + repository + service + handler + 前端组件 + 枚举定义）。
2. 业务规则必须具体、可验证：状态机迁移表、联动逻辑、校验边界、生命周期管理必须真实存在，禁止空壳 CRUD。
3. 本项目用于评测跨文件协同改动能力，禁止做成本目录、对账/财务、库存盘点、电商订单、预约挂号、工单客服、数据可视化报表类业务。
4. 前端页面必须真实消费后端接口，禁止纯静态假页面。

---
*生成说明：本提示词面向 Go 标注数据流水线 2000 行档位，主题已对照禁选题材清单核验。*
