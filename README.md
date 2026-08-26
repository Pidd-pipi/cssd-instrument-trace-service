# 消毒供应中心器械追溯服务（cssd-instrument-trace-service）

基于 Go 实现的消毒供应中心器械追溯 Web 项目（全栈 Web 应用）：Go 后端服务 + `go:embed` 内嵌原生 HTML/CSS/JS 前端，完成器械包登记、清洗消毒灭菌批次管理、发放回收闭环、灭菌参数合格判定与追溯查询。项目为干净基线，无故意埋错。

## 一、项目说明与核心领域规则

医院消毒供应中心（CSSD）负责手术器械包的清洗、消毒、灭菌、发放与回收。每个器械包有唯一条码，经历「回收→清洗→消毒→灭菌→发放→使用→回收」的循环。系统记录每个环节的操作人、设备、时间与关键参数，灭菌参数不合格的批次拦截不得发放，器械包按有效期管理，过期包禁止发放，整个循环可追溯。

1. **器械包循环状态机**：`to_collect → collected → washing → washed → sterilizing → sterilized → issued → in_use → to_collect`，循环必须按序推进，跳步拒绝；`expired` 只能进入 `washing`（重新处理）。
2. **灭菌参数判定**：每批次记录温度/时长/压力，与设定下限（默认 134℃ / 4min / 205kPa）比较，任一项不达标判定 `fail`，批次内器械包全部拦截退回「已清洗」。
3. **发放校验**：发放前强制校验「已灭菌 + 未过期 + 灭菌批次参数合格」三项，任一不满足拒绝（HTTP 422）。
4. **有效期管理**：灭菌完成后按包类型计算有效期（surgical 7 天 / dressing 14 天 / instrument 14 天 / implant 30 天），过期扫描任务按间隔将过期包置为 `expired`；过期包经「重新清洗」可再次灭菌发放。
5. **发放回收闭环**：发放记录绑定使用科室与手术间，回收时按条码闭环；未回收超过 24 小时进入「丢失待查」清单。
6. **追溯查询**：按器械包条码拉出完整循环记录（每环节时间、操作人、设备、参数），按灭菌批次查该批所有器械包去向。

## 二、架构与分层

```
请求 → middleware（requestID → access log → recover → security headers → audit）
  → httpapi（参数校验/统一响应） → service（业务编排） → store（内存仓储 + JSON 原子持久化）
```

- **domain**：领域实体、枚举、状态机、参数判定，不依赖外部层。
- **store**：内存 map 仓储，所有写操作在 `sync.RWMutex` 写锁内串行化，快照通过「临时文件 → fsync → rename」原子落盘；损坏文件备份 `.bak` 后降级为空库启动并告警。
- **service**：业务用例编排、发放/回收闭环、灭菌批次联动、过期扫描、审计、统计。
- **httpapi**：REST 路由、请求体/查询参数校验、统一响应 `{code,message,data}`、分页响应头。
- **middleware**：requestID、结构化访问日志、panic recover、基础安全头、审计中间件、发放守卫。
- **web**：原生 HTML/CSS/JS，`go:embed` 内嵌，无外部 CDN。

## 三、运行方式

### 本地运行

```bash
go run .                    # 默认监听 8080，数据文件 data/store.json
PORT=19005 go run .         # 指定监听端口
LOG_LEVEL=debug go run .    # 调整日志级别
```

启动后：

- `GET /healthz`、`GET /readyz` 返回 200
- `GET /` 可打开前端页面

### 测试与检查

```bash
go build ./...       # 构建全绿
go vet ./...         # 静态检查
gofmt -l .           # 格式检查（无输出表示通过）
go test ./...        # 单元/集成测试全绿
go test -race ./...  # 竞态检测全绿
```

### Docker 部署

```bash
docker build -t cssd-instrument-trace-service:latest .
docker run --rm -p 8080:8080 -e PORT=8080 -v "$PWD/data:/app/data" cssd-instrument-trace-service:latest
```

镜像说明：

- 多阶段构建：`golang:1.23-alpine` → `alpine:3.20`
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`，静态可移植二进制
- 非 root 用户 `app` 运行
- `EXPOSE 8080`，尊重 `PORT` 环境变量
- 内置 `HEALTHCHECK` 探测 `/healthz`

### Makefile

```bash
make build
make vet
make test
make race
make docker-build
make docker-run
make clean
```

## 四、环境变量表

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | HTTP 监听端口，必须是 1-65535 |
| `DATA_FILE` | `data/store.json` | JSON 持久化文件路径 |
| `LOG_LEVEL` | `info` | 全局结构化日志级别：`debug`/`info`/`warn`/`error` |
| `HTTP_READ_TIMEOUT` | `15s` | 完整请求体读取超时（Go duration） |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | 请求头读取超时（Go duration） |
| `HTTP_WRITE_TIMEOUT` | `30s` | 响应写超时（Go duration） |
| `HTTP_IDLE_TIMEOUT` | `60s` | keep-alive 空闲超时（Go duration） |
| `SHUTDOWN_TIMEOUT` | `10s` | 优雅关闭最大等待时间（Go duration） |
| `STERILIZE_MIN_TEMP` | `134` | 灭菌温度下限（℃） |
| `STERILIZE_MIN_DURATION` | `4` | 灭菌时长下限（分钟） |
| `STERILIZE_MIN_PRESSURE` | `205` | 灭菌压力下限（kPa） |
| `LOST_TIMEOUT_HOURS` | `24` | 发放未回收丢失判定小时数 |
| `SWEEP_INTERVAL_MIN` | `60` | 过期扫描定时任务间隔（分钟） |

配置加载后统一调用 `Config.Validate()`，非法配置直接拒绝启动。

## 五、API 表

统一响应格式：`{"code":0,"message":"ok","data":...}`；业务错误 `code` 为对应 HTTP 状态码。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz`、`/readyz`、`/api/healthz` | 健康/就绪检查（200） |
| GET | `/` 及 `/packs`、`/sterilization`、`/issue`、`/trace` | 前端页面（SPA 回退） |
| POST | `/api/packs` | 登记器械包 |
| GET | `/api/packs` | 器械包列表：`stage/type/keyword/limit/offset` |
| GET | `/api/packs/{id}` | 器械包详情 |
| POST | `/api/packs/{id}/cycle` | 环节流转（状态机顺序校验） |
| POST | `/api/packs/{id}/issue` | 发放（三项校验，issueGuard 前置守卫） |
| POST | `/api/packs/{id}/collect` | 回收闭环 |
| GET | `/api/packs/{id}/trace` | 器械包循环追溯 |
| GET | `/api/trace?barcode=` | 按条码追溯 |
| POST | `/api/sterilizations` | 创建灭菌批次（录入参数） |
| POST | `/api/sterilizations/{id}/complete` | 完成灭菌（参数判定 + 器械包状态更新） |
| GET | `/api/sterilizations` | 批次列表：`limit/offset` |
| GET | `/api/sterilizations/{id}` | 批次详情 |
| GET | `/api/sterilizations/{id}/packs` | 批次器械包去向 |
| GET/POST | `/api/sterilizers` | 灭菌器列表/登记 |
| GET | `/api/issues` | 发放记录列表：`status/packId/limit/offset` |
| GET | `/api/lost` | 丢失待查清单 |
| GET | `/api/dashboard` | 工作台总览 |
| GET | `/api/audit-logs` | 审计日志列表：`limit/offset` |

### 分页约定

所有 list 接口支持 `limit`/`offset`：

- 默认 `limit=20`，最大 `limit=200`，`offset` 默认 `0`；
- 响应体保持数组，兼容前端；分页元数据通过响应头返回：
  - `X-Total-Count`：过滤后的总条数
  - `X-Limit`：本次实际生效的 `limit`
  - `X-Offset`：本次实际生效的 `offset`
- 非法 `limit`/`offset` 返回 400。

### 输入校验

- 请求体 JSON 仅允许一个对象，未知字段、超长请求体、畸形 JSON 均返回 400；
- 路径参数 `id` 缺失/为空返回 400；
- 非法 `stage`、`type`、`status` 查询参数返回 400；
- 领域校验（条码/包名/包类型、灭菌器、发放三要素、状态机顺序）由 service/domain 层执行，错误映射为 400/404/409/422，不会 panic。

## 六、目录结构

```
cssd-instrument-trace-service/
├── go.mod / main.go          # 入口：go:embed 前端、配置校验、结构化日志、全超时服务与优雅关闭
├── Dockerfile                # 多阶段、非 root、HEALTHCHECK
├── .dockerignore
├── Makefile
├── runtime_smoke.json        # 冒烟契约（保留不变）
├── config/                   # 配置与业务规则
│   ├── config.go             # 环境变量加载、Config.Validate
│   └── rules.go              # 灭菌下限、有效期、回收时限
├── domain/                   # 领域实体 + 状态机 + 枚举
│   ├── constants.go          # PackStage / SterilizeResult / PackType 等枚举
│   ├── pack_stage.go         # 状态机迁移表 + 校验
│   ├── pack.go / sterilization.go / cycle.go / issue.go / sterilizer.go / audit.go
│   ├── errors.go / ids.go
├── store/                    # 内存仓储 + JSON 原子持久化
│   ├── store.go / persistence.go / seed.go
│   ├── pack_store.go / sterilization_store.go / cycle_store.go
│   ├── issue_store.go / sterilizer_store.go / audit_store.go
├── service/                  # 业务用例编排
│   ├── service.go / errors.go / audit_service.go
│   ├── pack_service.go / sterilization_service.go / issue_service.go
│   ├── trace_service.go / expiry_sweeper.go / stats_service.go
├── httpapi/                  # REST 接口层
│   ├── router.go / response.go / pagination.go / params.go
│   ├── pack_handler.go / sterilization_handler.go / issue_handler.go
│   ├── trace_handler.go / dashboard_handler.go / audit_handler.go / health_handler.go
├── middleware/               # 横切关注点
│   ├── request_id.go / request_log.go / error_handler.go / security.go
│   ├── audit.go / issue_guard.go / status.go
└── web/                      # go:embed 内嵌前端（原生 HTML/CSS/JS）
    ├── index.html / app.js / api.js / constants.js / style.css
    ├── components/           # StageBadge / CycleTimeline / IssueForm
    ├── hooks/                # usePacks / useTrace
    └── pages/                # dashboard / packs / sterilization / issue / trace
```

## 七、枚举/常量前后端出现位置

| 枚举/常量 | 后端定义 | 前端定义 |
|---|---|---|
| PackStage（to_collect/collected/washing/washed/sterilizing/sterilized/issued/in_use/expired） | `domain/constants.go`、`domain/pack_stage.go` | `web/constants.js`（`PackStage`、`manualCycle`） |
| SterilizeResult（pass/fail） | `domain/constants.go` | `web/constants.js`（`SterilizeResult`） |
| PackType（surgical/dressing/instrument/implant） | `domain/constants.go` | `web/constants.js`（`PackType`） |
| IssueStatus / SterilizerStatus / BatchStatus | `domain/constants.go` | `web/constants.js` |
| 灭菌参数下限 | `config/rules.go` | `web/constants.js`（`limits`，仅提示展示） |
| 审计动作常量 | `domain/audit.go` | —（语义由前端页面展示） |

## 八、共享前端组件与 hooks

- **components/**：`StageBadge`（工作台 + 器械包列表 + 批次去向共用）、`CycleTimeline`（追溯页 + 器械包详情共用）、`IssueForm`（发放页 + 器械包详情共用）。
- **hooks/**：`usePacks(filter)`（器械包列表，列表页 + 工作台共用）、`useTrace(id)`（循环追溯，追溯页 + 详情共用）。

## 九、可复现的业务链路（冒烟示例）

1. `POST /api/packs` 登记器械包（初始 `to_collect`）；
2. `POST /api/packs/{id}/cycle` 按序推进 `collected → washing → washed`（跳步返回 409）；
3. `POST /api/sterilizations` 创建批次（装载「已清洗」器械包 → `sterilizing`）；
4. `POST /api/sterilizations/{id}/complete` 参数判定（达标 → `sterilized` 并计算有效期；不达标 → 退回 `washed` 整批拦截）；
5. `POST /api/packs/{id}/issue` 发放（三项校验，任一不满足 422）；
6. `POST /api/packs/{id}/cycle` 标记 `in_use`；
7. `POST /api/packs/{id}/collect` 回收闭环（回到 `to_collect`）；
8. `GET /api/packs/{id}/trace`、`GET /api/sterilizations/{id}/packs` 追溯查询；
9. `GET /api/lost`、`GET /api/dashboard` 总览与丢失清单。

## 十、健康检查与故障排查

| 现象 | 排查 |
|---|---|
| `go run .` 启动失败 | 查看环境变量是否非法；确认 `DATA_FILE` 目录可写；`PORT` 未占用 |
| `/healthz` 非 200 | 查看 stdout 结构化日志，确认服务是否启动、`DATA_FILE` 是否可读写 |
| 数据文件损坏 | 启动时自动备份为 `data/store.json.bak`（或带时间戳后缀）并降级为空库，日志输出 `WARN` |
| 接口返回 400 | 检查请求体是否为单个合法 JSON 对象、路径/查询参数是否非法 |
| 接口返回 409 | 状态机跳步、重复条码或批次重复完结，按业务规则修正 |
| 接口返回 422 | 发放/回收守卫拦截，查看 `message` 中的具体原因 |
| 日志需要更详细 | 设置 `LOG_LEVEL=debug` |
| 前端页面不可用 | 确认访问 `/`，若本地 curl 可返回 HTML；项目不依赖 CDN，离线可运行 |

## 十一、质量说明

- 分层架构严格：domain / store / service / httpapi / middleware / web，禁止逻辑堆叠在 main.go 或单个 handlers.go。
- 持久化采用临时文件 + `fsync` + `os.Rename` 原子替换，写操作受 `sync.RWMutex` 串行化保护。
- HTTP 服务配置 ReadTimeout / ReadHeaderTimeout / WriteTimeout / IdleTimeout，并实现 SIGINT/SIGTERM 优雅关闭。
- 全局结构化日志 `log/slog`，`LOG_LEVEL` 控制；请求日志、panic 恢复、损坏文件告警均结构化输出。
- `go build ./...`、`go vet ./...`、`gofmt -l .`、`go test ./...`、`go test -race ./...` 全绿。
