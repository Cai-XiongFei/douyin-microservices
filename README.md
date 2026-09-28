# Douyin 微服务后端

这是一个使用 Go 实现的短视频后端学习项目，包含用户、视频、点赞、评论、关注关系和即时消息等业务。项目使用 Hertz 提供 HTTP API，使用 Kitex 完成 RPC 通信，并结合 MySQL、Redis、RabbitMQ、etcd、MinIO、FFmpeg 和 Nginx 实现数据存储、缓存、可靠消息、服务发现、媒体处理与负载均衡。

## 一、项目特性

- Hertz HTTP API 和中间件。
- Kitex、Protobuf RPC 微服务。
- etcd 服务注册、发现与 RPC 负载均衡。
- GORM、MySQL 事务、索引和读写路由。
- Redis Cache Aside 缓存、分布式锁和 Lua 令牌桶限流。
- RabbitMQ Publisher Confirm、手动 ACK、重试队列和死信队列。
- 点赞业务使用 Transactional Outbox，保证业务数据与待发事件原子写入。
- 消费者通过事件 ID 实现幂等，支持消息至少一次投递。
- MinIO 对象存储和 FFmpeg 视频封面抽取。
- Nginx 多 API 实例负载均衡。
- Docker Compose 基础设施和 PowerShell 启停脚本。

## 二、系统架构

~~~mermaid
flowchart LR
    Client[Postman / 客户端] --> Nginx[Nginx :18080]
    Nginx --> API1[Hertz API :18089]
    Nginx --> API2[Hertz API :18099]
    Nginx --> API3[Hertz API :18109]

    API1 --> Etcd[etcd 服务发现]
    API2 --> Etcd
    API3 --> Etcd

    Etcd --> User[User RPC :18085]
    Etcd --> Video[Video RPC :18086]
    Etcd --> Favorite[Favorite RPC :18087]
    Etcd --> Comment[Comment RPC :18088]
    Etcd --> Relation[Relation RPC :18090]
    Etcd --> Message[Message RPC :18091]

    User --> MySQL[(MySQL)]
    Video --> MySQL
    Favorite --> MySQL
    Comment --> MySQL
    Relation --> MySQL
    Message --> MySQL

    User --> Redis[(Redis)]
    Video --> Redis
    Favorite --> Redis
    Video --> MinIO[(MinIO)]
    Video --> FFmpeg[FFmpeg]

    Favorite --> Outbox[(Outbox 表)]
    Outbox --> Relay[Outbox Relay]
    Relay --> RabbitMQ[(RabbitMQ)]
    RabbitMQ --> Consumers[事件消费者]
    Consumers --> Redis
~~~

主要请求链路：

~~~text
客户端
  → Hertz 路由和中间件
  → Kitex RPC 客户端
  → etcd 发现 RPC 实例
  → RPC Service
  → GORM / Redis / MinIO
  → RPC 响应
  → HTTP JSON 响应
~~~

## 三、技术栈

| 分类 | 技术 | 作用 |
| --- | --- | --- |
| 语言 | Go 1.19 | 服务端开发 |
| HTTP | CloudWeGo Hertz | API、路由和中间件 |
| RPC | CloudWeGo Kitex | 微服务通信 |
| IDL | Protobuf | 定义 RPC 输入、输出和服务 |
| 注册中心 | etcd | 服务注册、发现和负载均衡 |
| 数据库 | MySQL 8 | 业务数据和 Outbox |
| ORM | GORM | 模型、查询、事务和迁移 |
| 缓存 | Redis 7 | 缓存、锁、限流和幂等 |
| 消息队列 | RabbitMQ 3.13 | 异步事件、重试和死信 |
| 对象存储 | MinIO | 视频、封面、头像和背景图 |
| 媒体处理 | FFmpeg | 视频抽帧生成封面 |
| 网关 | Nginx | 反向代理和 HTTP 负载均衡 |
| 配置 | Viper | YAML 配置读取 |
| 鉴权 | JWT | 登录身份认证 |
| 容器 | Docker Compose | 本地基础设施 |

## 四、目录结构

~~~text
douyin/
├── cmd/
│   ├── api/                 # Hertz API、Handler 和 RPC 客户端
│   ├── user/                # 用户 RPC
│   ├── video/               # 视频 RPC
│   ├── favorite/            # 点赞 RPC、消费者和 Outbox Relay
│   ├── comment/             # 评论 RPC 和消费者
│   ├── relation/            # 关系 RPC 和消费者
│   └── message/             # 消息 RPC 和消费者
├── config/                  # 应用和中间件配置
├── dal/
│   ├── db/                  # GORM 模型和 MySQL 数据访问
│   └── redis/               # Redis 客户端和底层操作
├── deploy/nginx/            # Nginx 配置
├── internal/
│   ├── cache/               # Cache Aside 业务缓存
│   ├── limiter/             # Redis Lua 令牌桶
│   ├── lock/                # Redis 分布式锁
│   ├── outbox/              # Outbox Relay
│   ├── response/            # HTTP 响应结构
│   └── tool/                # 密码摘要、视频封面等工具
├── kitex/
│   ├── *.proto              # RPC 接口定义
│   └── kitex_gen/           # Kitex 生成代码，不建议手动修改
├── pkg/
│   ├── etcd/                # 服务注册与发现
│   ├── jwt/                 # JWT 创建和解析
│   ├── middleware/          # 鉴权、日志和限流
│   ├── minio/               # MinIO 初始化和对象操作
│   ├── rabbitmq/            # 发布、消费、重试和死信
│   └── viper/               # 配置封装
├── scripts/
│   ├── start.ps1            # 构建并启动全部服务
│   └── stop.ps1             # 停止脚本管理的服务
├── compose.yaml
└── go.mod
~~~

## 五、端口说明

### 应用服务

| 服务 | 端口 | 说明 |
| --- | ---: | --- |
| User RPC | 18085 | 注册、登录、用户信息 |
| Video RPC | 18086 | 发布、作品列表、Feed |
| Favorite RPC | 18087 | 点赞、喜欢列表 |
| Comment RPC | 18088 | 评论操作、评论列表 |
| API | 18089 | Hertz HTTP API |
| Relation RPC | 18090 | 关注、粉丝、好友 |
| Message RPC | 18091 | 发消息、聊天记录 |
| API 实例 2 | 18099 | Nginx 多实例测试 |
| API 实例 3 | 18109 | Nginx 多实例测试 |

### Docker 基础设施

| 组件 | 本机端口 | 容器端口 | 说明 |
| --- | ---: | ---: | --- |
| MySQL source | 3309 | 3306 | 写入口 |
| MySQL replica1 | 3310 | 3306 | 本地模拟读入口 |
| MySQL replica2 | 3308 | 3306 | 本地模拟读入口 |
| etcd client | 12379 | 2379 | 服务注册和发现 |
| MinIO API | 19000 | 9000 | 对象访问 |
| MinIO Console | 19001 | 9001 | 管理界面 |
| Redis | 16379 | 6379 | 缓存、锁和限流 |
| RabbitMQ AMQP | 15672 | 5672 | 应用连接端口 |
| RabbitMQ Console | 15673 | 15672 | 管理界面 |
| Nginx | 18080 | 80 | HTTP 统一入口 |

三个 MySQL 本机端口映射到同一个容器，只用于学习 GORM dbresolver 的读写路由，并不是真正的主从集群。

管理界面：

- MinIO：http://127.0.0.1:19001，账号和密码均为 tiktokMinio。
- RabbitMQ：http://127.0.0.1:15673，账号 tiktok，密码 tiktokRabbitMQ。

## 六、环境要求

- Windows 10/11 和 PowerShell。
- Go 1.19，项目开发环境使用 Go 1.19.13。
- Docker Desktop，并确保 Linux Container Engine 已启动。
- FFmpeg，并确保 ffmpeg 命令在 PATH 中。
- 可选：Postman。
- 只有重新生成 RPC 代码时才需要 protoc、protoc-gen-go 和 kitex。

~~~powershell
go version
docker --version
docker compose version
ffmpeg -version
~~~

## 七、快速启动

进入项目并下载依赖：

~~~powershell
cd D:\project\tiktok\douyin
go mod tidy
~~~

### 方式一：使用脚本

脚本会启动 Docker 基础设施、构建服务、先启动 RPC、等待端口就绪，再启动 API。日志保存在 logs/，PID 保存在 .run/。

~~~powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\start.ps1
~~~

已有 bin/ 可执行文件时跳过构建：

~~~powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\start.ps1 -SkipBuild
~~~

验证：

~~~powershell
Invoke-RestMethod -Method Get -Uri "http://127.0.0.1:18089/healthz"
~~~

预期响应：

~~~json
{
  "status_code": 0,
  "status_msg": "success",
  "service": "douyin-api服务"
}
~~~

停止应用但保留 Docker：

~~~powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\stop.ps1
~~~

同时停止基础设施：

~~~powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\stop.ps1 -StopInfrastructure
~~~

### 方式二：手动启动

~~~powershell
docker compose up -d
docker compose ps
~~~

分别打开终端启动 RPC：

~~~powershell
go run ./cmd/user
go run ./cmd/video
go run ./cmd/favorite
go run ./cmd/comment
go run ./cmd/relation
go run ./cmd/message
~~~

最后启动 API：

~~~powershell
go run ./cmd/api
~~~

API 初始化时需要通过 etcd 找到 RPC 实例，因此应先启动 RPC，再启动 API。

## 八、Nginx 多实例

普通开发直接访问 http://127.0.0.1:18089。

Nginx 默认配置 18089、18099、18109 三个 API 实例。使用 Nginx 前应全部启动，否则轮询到未启动实例时可能返回 502。

启动脚本已经启动 18089，再打开两个终端：

~~~powershell
$env:API_PORT="18099"
go run ./cmd/api
~~~

~~~powershell
$env:API_PORT="18109"
go run ./cmd/api
~~~

通过 Nginx 访问：

~~~powershell
Invoke-WebRequest -Uri "http://127.0.0.1:18080/healthz" | Select-Object StatusCode,Headers
~~~

响应头 X-Upstream-Addr 显示实际处理请求的 API 实例。

## 九、HTTP API

默认地址：http://127.0.0.1:18089

JWT 中间件当前从 Query 参数或表单字段读取 token，不从 Authorization 请求头读取。

| 方法 | 路径 | 鉴权 | 主要参数 | 说明 |
| --- | --- | --- | --- | --- |
| GET | /healthz | 否 | 无 | 健康检查 |
| POST | /douyin/user/register/ | 否 | username、password | 注册 |
| POST | /douyin/user/login/ | 否 | username、password | 登录 |
| GET | /douyin/user/ | 是 | user_id、token | 用户信息 |
| GET | /douyin/feed/ | 否 | latest_time、可选 token | 视频流 |
| POST | /douyin/publish/action/ | 是 | multipart：data、title、token | 发布视频 |
| GET | /douyin/publish/list/ | 是 | user_id、token | 作品列表 |
| POST | /douyin/favorite/action/ | 是 | video_id、action_type、token | 点赞/取消 |
| GET | /douyin/favorite/list/ | 是 | user_id、token | 喜欢列表 |
| POST | /douyin/comment/action/ | 是 | video_id、action_type、comment_text 或 comment_id、token | 发布/删除评论 |
| GET | /douyin/comment/list/ | 否 | video_id、可选 token | 评论列表 |
| POST | /douyin/relation/action/ | 是 | to_user_id、action_type、token | 关注/取消 |
| GET | /douyin/relation/follow/list/ | 是 | user_id、token | 关注列表 |
| GET | /douyin/relation/follower/list/ | 是 | user_id、token | 粉丝列表 |
| GET | /douyin/relation/friend/list/ | 是 | user_id、token | 好友列表 |
| POST | /douyin/message/action/ | 是 | to_user_id、action_type、content、token | 发送消息 |
| GET | /douyin/message/chat/ | 是 | to_user_id、pre_msg_time、token | 聊天记录 |

action_type 通常使用：

- 点赞、关注、发布评论：1。
- 取消点赞、取消关注、删除评论：2。

## 十、接口测试

注册、登录并保存 token：

~~~powershell
$baseURL = "http://127.0.0.1:18089"
Invoke-RestMethod -Method Post -Uri "$baseURL/douyin/user/register/?username=demo_user&password=123456"
$login = Invoke-RestMethod -Method Post -Uri "$baseURL/douyin/user/login/?username=demo_user&password=123456"
$token = [Uri]::EscapeDataString($login.token)
$userID = $login.user_id
~~~

查询用户和 Feed：

~~~powershell
Invoke-RestMethod -Method Get -Uri "$baseURL/douyin/user/?user_id=$userID&token=$token"
Invoke-RestMethod -Method Get -Uri "$baseURL/douyin/feed/" | ConvertTo-Json -Depth 10
~~~

点赞：

~~~powershell
Invoke-RestMethod -Method Post -Uri "$baseURL/douyin/favorite/action/?video_id=1&action_type=1&token=$token"
~~~

点赞接口启用了 Redis 令牌桶。当前容量为 3，每 10 秒补充约 1 个令牌；请求过快返回 HTTP 429，并携带 X-RateLimit-Limit 和 X-RateLimit-Remaining。

发布视频：

~~~powershell
curl.exe -X POST -F "data=@D:\videos\demo.mp4" -F "title=测试视频" "http://127.0.0.1:18089/douyin/publish/action/?token=$token"
~~~

在 Postman 中应把方法选为 POST，URL 框只填写 URL，不要把 “POST ” 文本一起粘贴进去。

## 十一、核心设计

### Transactional Outbox

点赞业务在同一个 MySQL 事务内完成：

~~~text
点赞关系变化
+ 视频和用户计数变化
+ 写入 pending Outbox 事件
= 同一个数据库事务
~~~

状态流转：

~~~text
pending → processing → sent
                    ↘ failed
~~~

- FOR UPDATE SKIP LOCKED 支持多个 Relay 并行领取。
- locked_by 和 locked_until 实现租约和崩溃接管。
- RabbitMQ Publisher Confirm 确认消息已被代理接收。
- 失败后指数退避，超过上限进入 failed。
- sent 不代表消费者已经处理完成。
- Relay 是至少一次投递，消费者必须幂等。

查看最近事件：

~~~powershell
docker exec douyin-mysql mysql -utiktokDB -ptiktokDB db_tiktok -e "SELECT id,event_type,status,retry_count,last_error,created_at,sent_at FROM outbox_events ORDER BY created_at DESC LIMIT 10;"
~~~

### RabbitMQ 可靠消费

~~~text
主队列处理失败
  → 重试队列延迟
  → 返回主队列
  → 超过次数进入死信队列
~~~

消息使用持久化、Publisher Confirm 和手动 ACK。复制到重试或死信队列并获得确认后，才 ACK 原消息。

### Redis 缓存与分布式锁

~~~text
读取缓存
  → 命中：直接返回
  → 未命中：查询 MySQL
             → 获取分布式锁
             → 重建缓存
~~~

锁使用 SET NX EX 获取，释放时通过 Lua 原子执行“比较唯一令牌并删除”，避免误删其他请求后来获得的锁。

### Redis 分布式限流

点赞接口使用 Redis Lua 令牌桶，多个 API 实例共享限流状态。Redis 故障时采用 fail-open：记录错误但允许业务继续。

### 数据库事务和读写路由

点赞、评论和关注等业务使用事务维护关系与计数一致性，唯一索引防止重复点赞和关注。dbresolver 配置了写入口和读入口，但本地三个端口仍指向同一个 MySQL。

## 十二、配置

- config/api.yml：API 地址、请求体限制、JWT、etcd、TLS/HTTP2。
- config/db.yml：MySQL source 和 replica。
- config/redis.yml：Redis。
- config/rabbitmq.yml：RabbitMQ 和消费重试。
- config/minio.yml：MinIO、Bucket 和临时 URL。
- config/各服务.yml：RPC 服务名、地址、端口和 JWT。
- config/favorite.yml：Outbox 批量、轮询、租约和重试参数。

生产环境应使用环境变量或密钥系统注入密码和 JWT 密钥，并为数据库和中间件配置高可用、备份、TLS 和网络访问控制。

## 十三、测试

~~~powershell
docker compose up -d
docker compose ps
~~~

仅编译全部包：

~~~powershell
go test ./... -run '^$' -count=1
~~~

运行全部测试：

~~~powershell
go test ./... -count=1
~~~

运行指定模块：

~~~powershell
go test ./dal/db -count=1 -v
go test ./dal/redis -count=1 -v
go test ./internal/limiter -count=1 -v
go test ./internal/outbox -count=1 -v
go test ./pkg/rabbitmq -count=1 -v
~~~

部分测试是集成测试，会连接 MySQL、Redis、RabbitMQ 和 MinIO，并可能创建后再清理测试数据。

## 十四、常见问题

### Docker 无法连接 Linux Engine

以下错误通常表示 Docker Desktop 没有启动：

~~~text
open //./pipe/dockerDesktopLinuxEngine: The system cannot find the file specified
~~~

~~~powershell
docker info
docker compose up -d
~~~

### 端口被占用

~~~powershell
Get-NetTCPConnection -LocalPort 18087 -State Listen | Select-Object LocalAddress,LocalPort,OwningProcess
~~~

确认进程属于本项目后再执行：

~~~powershell
Stop-Process -Id <PID>
~~~

### API 返回 no instance found

检查 etcd、对应 RPC 服务，以及 RPC 注册名称和 API 发现名称是否一致。

### MinIO 连接被拒绝

~~~powershell
docker compose up -d minio
docker compose ps minio
Test-NetConnection 127.0.0.1 -Port 19000
~~~

### FFmpeg 找不到

将 FFmpeg 的 bin 目录加入 PATH，然后重新打开 VS Code 和终端：

~~~powershell
where.exe ffmpeg
ffmpeg -version
~~~

### Nginx 偶尔返回 502

默认 upstream 有三个 API 地址，请同时启动 18089、18099、18109，或者只保留实际运行的 upstream。

## 十五、当前边界与后续规划

当前边界：

- 尚未接入 Prometheus、Grafana 和分布式链路追踪。
- 密码摘要需要升级为 bcrypt 或 Argon2 后才能用于生产。
- 配置中的开发密钥和密码不能用于生产。
- 本地 MySQL replica 不是真实主从复制。
- Outbox 当前重点应用于点赞业务，其他事件可逐步迁移。
- 尚未实现真实分库分表。

后续计划：

- 接入 Prometheus、Grafana 和 OpenTelemetry。
- 增加 Outbox 积压、failed 事件和 RabbitMQ 队列深度告警。
- 将更多业务事件迁移到 Transactional Outbox。
- 增加压力测试，记录 QPS、P95/P99 延迟和缓存命中率。
- 增加 CI、静态检查、镜像构建和自动部署。
