# dao-on-gorm

一个基于 gorm 的**单行 DAO** 封装：一个 struct 绑定一行数据，封装 `Load / Create / Update /
Save / Delete`，并在主键维度做自动缓存、自动填充 `created_at` / `updated_at`。

包名 `daoongorm`，模块路径 `github.com/cclehui/dao-on-gorm`。这是一个被多个业务服务引用的库，
不是可部署的服务：**向后兼容优先**，任何会让既有 DAO 行为静默变化的改动都要当成 breaking change 处理。

先读与任务匹配的文档，不要一次加载全部：

| 读这个 | 什么时候 |
|---|---|
| [harness/coding_style.md](./harness/coding_style.md) | 写/评审代码前：文件职责、命名、注释、错误、日志。 |
| [harness/knowledge/architecture.md](./harness/knowledge/architecture.md) | 想搞清楚 `DaoBase` / `Model` / `DBClient` / `CacheInterface` 各层边界和一次调用的完整链路。 |
| [harness/knowledge/data_model.md](./harness/knowledge/data_model.md) | 定义或修改 DAO 结构体、gorm tag、主键、时间列精度；排查“模型未注册 / 字段没生效”。 |
| [harness/knowledge/crud.md](./harness/knowledge/crud.md) | 需要确认 Load/Create/Update/Save/Delete 的精确语义（部分更新、时间填充、事务加锁、缓存时机）。 |
| [harness/knowledge/cache.md](./harness/knowledge/cache.md) | 动缓存 key、TTL、失效策略，或接自定义 `CacheInterface`。 |
| [harness/knowledge/db_client.md](./harness/knowledge/db_client.md) | 动连接、读写分离、事务、超时、日志级别配置。 |
| [harness/knowledge/testing.md](./harness/knowledge/testing.md) | 跑测试、加测试，或需要用一次性 MySQL/redis 验证真实行为。 |
| [harness/knowledge/sql_injection.md](./harness/knowledge/sql_injection.md) | 写 where / 排序 / 动态表名，或怀疑某段 SQL 拼接有注入风险。 |

## 环境

- Go `1.21`（`go.mod` 声明；本地用更新的工具链也可以）。
- `gorm.io/gorm v1.31.2` + `gorm.io/driver/mysql v1.6.0`，`go-sql-driver/mysql v1.8.1`。
  这三个版本一起升级，不要只升其中一个——driver 与 gorm 的 `precision` / DDL 行为是耦合的。
- 测试分两类：**纯单元测试**（不连库，默认都跑）和 **examples 集成测试**（需要真实 MySQL + redis）。

```bash
# 单元测试（不需要任何外部依赖）
go build ./... && go vet ./... && go test .

# examples 是独立 module（examples/go.mod），不在根模块的 ./... 里
cd examples && go build ./... && go vet ./...   # 编译检查
cd examples && go test ./...                    # 集成测试（需要 MySQL + redis）
```

`examples` 独立成 module 是有意的：redis / testify / yaml 只有 examples 用，不能污染根
`go.mod`——下游 `go get` 本库时不该被带动引入这些依赖。新增 examples 依赖时改
`examples/go.mod`，不要加回根模块。

## 提交需要明确指令

除非用户在**当次请求**里明确要求（例如“commit”“commit and push”），否则不要执行
`git commit` / `git push` / `git add`。

- 完成一个任务、测试通过，都**不构成**提交许可。
- 权限只覆盖一次提交；下一个任务需要新的指令。
- 结束时把改动留在工作区，并报告状态。
- `git status / diff / log / show` 这类只读操作随时可以执行。

## 不要随便改的行为（兼容性红线）

这些是已经对外承诺、被 examples 和下游服务依赖的语义。改之前先和用户确认，并同步更新
`harness/knowledge/crud.md` / `cache.md`：

- `Save` 在主键为空时走 `Create`，否则走 `Update`；`IsNewRow()` 的取值时机不能变。
- 未显式修改 `updated_at` 时，`Save` 会把它 touch 到当前时间（原库约定，examples 依赖）。
- 只读 DAO（`readOnly=true` / `NewDaoBaseNoLoad`）上的写操作必须报错，不能静默降级。
- 缓存 key 的默认格式 `daocache:<db>:<table>:<pk>` 不能变，否则升级会让线上缓存整体失效。
- 事务内不能把未提交的数据写进缓存。

## 安全与正确性不变量

- **ctx 必须传到 gorm**：统一用 `WithContext(ctx)`，不要再用 `Set("scope_context", ctx)`。
  否则请求取消、deadline、链路追踪都传不到 SQL 层（这是历史 bug，已修）。
- **事务内读单行要加锁**：`Load` 在事务里用 `clause.Locking{Strength: "UPDATE"}`。
- **缓存失效只能发生在 COMMIT 之后**：事务内注册到 `DBClient.AddAfterCommit`，回滚时不得失效。
- **API key 不进 SQL 拼接**：`cacheKey` / where 条件都用绑定参数；`PKFieldsStr` 只用于缓存 key。
- **值只能走绑定参数**：`Where("col = ?", v)`，禁止 `+` / `fmt.Sprintf` 拼值。标识符（列名、
  表名、`ORDER BY`）不能用 `?`，只能来自白名单或代码常量。详见
  [harness/knowledge/sql_injection.md](./harness/knowledge/sql_injection.md)。
- **时间精度按列声明**，不要引入全局精度开关：不同表的 `datetime(n)` 不同。
- 不允许打印或缓存密钥类字段（`client_secret`、证书私钥等）；缓存值即整行结构体，定义 DAO 时注意。

## 文档维护

- `harness/knowledge/` 放**事实**（行为、契约、为什么这么设计），`harness/coding_style.md` 放**约束**。
- 改动行为时，同一提交内更新对应 knowledge 文件；不要留下“代码已变、文档还写老行为”。
- 注释用中文、只写“为什么”，不复述代码。

## 本地验证用的容器

本机 Docker 已缓存 `mysql:8.0.44` 与 `redis:5.0.14`，验证 SQL / 缓存行为时**优先用本地镜像**，
不要直接 `docker pull`（外网拉取经常超时）：

```bash
docker run -d --rm --name daoongorm-it \
  -e MYSQL_ROOT_PASSWORD=123456 -e MYSQL_DATABASE=test \
  -e MYSQL_USER=cclehui_test -e MYSQL_PASSWORD=123456 -p 13306:3306 mysql:8.0.44

docker run -d --rm --name daoongorm-redis \
  -p 6379:6379 redis:5.0.14 redis-server --requirepass 123456
```

用完 `docker stop daoongorm-it daoongorm-redis`。不要指向任何真实业务库。
