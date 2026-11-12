# dao-on-gorm

[English](./README.en.md) | 简体中文

[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

`dao-on-gorm` 是构建在 [gorm](https://gorm.io/) 之上的**单行 DAO** 封装：

> 一个 struct 绑定一行数据，`DaoBase` 负责按主键完成 `Load / Create / Update / Save / Delete`，
> 并自动处理主键维度的缓存与 `created_at` / `updated_at` 填充。

调用方不再手写 `WHERE` 条件，也不需要在业务代码里散落缓存逻辑。

```go
user, err := NewUserDao(ctx, &User{ID: 1}, false)
if err != nil {
    return err
}

user.Name = "connor"
return user.GetDaoBase().Save(ctx) // 自动识别 Insert / Update
```

## 特性

- **单行语义**：一个 DAO = 一行，按主键读写，不涉及列表/条件查询。
- **构造即加载**：`NewDaoBase` 自动 `Load`，构造成功后 `IsNewRow()` 立即可用。
- **自动时间戳**：`created_at` / `updated_at` 零值时自动填充，并支持按列精度归一化。
- **部分更新**：`Save` 只更新真正变化的非主键列，避免并发场景互相覆盖。
- **主键缓存**：读单行优先命中缓存，写操作自动失效；默认不缓存，可插拔 `CacheInterface`。
- **事务安全**：事务内 `FOR UPDATE` 锁行，缓存失效延迟到 `COMMIT` 之后。
- **读写分离**：`DBClient` 内置只读副本轮询与事务 clone，DAO 层无感知。
- **可扩展**：`CacheInterface` / `DBClientInterface` 均可自定义实现。

## 安装

```bash
go get github.com/cclehui/dao-on-gorm
```

要求 Go `1.21+`，`gorm.io/gorm v1.31.2` 与 `gorm.io/driver/mysql v1.6.0`。

## 快速开始

### 1. 定义模型

```go
type User struct {
    ID        int64     `gorm:"column:id;primaryKey" structs:"id" json:"id"`
    Name      string    `gorm:"column:name" structs:"name" json:"name"`
    CreatedAt time.Time `gorm:"column:created_at;precision:3" structs:"created_at" json:"created_at"`
    UpdatedAt time.Time `gorm:"column:updated_at;precision:3" structs:"updated_at" json:"updated_at"`

    daoBase *daoongorm.DaoBase
}

func (u *User) TableName() string { return "users" }
func (u *User) DBName() string    { return "test" }

func (u *User) DBClient() daoongorm.DBClientInterface { return GetDBClient() }
func (u *User) GetDaoBase() *daoongorm.DaoBase        { return u.daoBase }
func (u *User) SetDaoBase(base *daoongorm.DaoBase)    { u.daoBase = base }
```

### 2. 注册模型

模型必须在进程启动阶段注册，`NewDaoBase` 依赖注册期解析出的元数据：

```go
func init() {
    daoongorm.RegisterModel(&User{})
}
```

未注册的模型会直接返回 `模型未注册, <type>`；缺少主键、列名重复等编码错误会在注册期 panic。

### 3. 构造入口

```go
func NewUserDao(ctx context.Context, user *User, readOnly bool,
    options ...daoongorm.Option) (*User, error) {

    base, err := daoongorm.NewDaoBase(ctx, user, readOnly, options...)
    user.daoBase = base
    return user, err
}

func NewUserDaoWithTX(ctx context.Context, user *User, tx *daoongorm.DBClient,
    options ...daoongorm.Option) (*User, error) {

    base, err := daoongorm.NewDaoBaseWithTX(ctx, user, tx, options...)
    user.daoBase = base
    return user, err
}
```

`NewDaoBase` 构造时会立即 `Load`：

| 入参 | 行为 |
|---|---|
| 主键为空 | 不查库，按 `column_default` 填充默认值，`IsNewRow() == true` |
| 主键非空 + `readOnly=true` | 优先读缓存，未命中读只读副本 |
| 主键非空 + `readOnly=false` | 读主库；事务内使用 `FOR UPDATE` |

### 4. 增删改查

```go
ctx := context.Background()

// Create：主键为空 → Save 走 Insert
user, err := NewUserDao(ctx, &User{Name: "connor"}, false)
if err != nil {
    return err
}
err = user.GetDaoBase().Save(ctx)
// user.ID 已回填，IsNewRow() == false

// Load：按主键读取（readOnly=true 走缓存 + 只读副本）
user, err = NewUserDao(ctx, &User{ID: 1}, true)

// Update：只提交变化列，updated_at 自动 touch
user.Name = "new name"
err = user.GetDaoBase().Save(ctx)

// Delete：按主键物理删除，并失效缓存
err = user.GetDaoBase().Delete(ctx)
```

> `Save` 在未显式修改 `updated_at` 时会将其更新为当前时间，因此即使业务字段未变也会产生一条只更新
> `updated_at` 的 SQL —— 这是本库既有约定。

### 5. 事务

```go
err := dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
    user, err := NewUserDaoWithTX(ctx, &User{ID: 1}, tx)
    if err != nil {
        return err
    }

    user.Name = "connor"
    return user.GetDaoBase().Save(ctx)
})
```

- 事务内**必须**使用 `NewDaoBaseWithTX` / `NewXxxDaoWithTX`，否则会走主库非事务连接，不受回滚保护。
- 事务内读取单行会加 `FOR UPDATE` 行锁。
- 缓存失效注册到 `AddAfterCommit`，`COMMIT` 成功后才执行；回滚时缓存保持原值。

## 缓存

缓存按主键保存**整行结构体**，只作用于“按主键读单行”这条路径，列表与条件查询不经过该缓存。
默认实现为 `NopCacheUtil`，即不缓存。

```go
type CacheInterface interface {
    Set(ctx context.Context, key string, value interface{}, ttl int) error
    Get(ctx context.Context, key string, value interface{}) (hit bool, error)
    Del(ctx context.Context, key string) error
}
```

全局注入或按 DAO 覆盖：

```go
daoongorm.SetGlobalCacheUtil(myCache)

user, err := NewUserDao(ctx, &User{ID: 1}, true,
    daoongorm.OptionSetCacheUtil(myCache),
    daoongorm.OptionSetCacheExpireTS(3600),
)
```

**Key 结构**：`[package_prefix:]daocache:<db>:<table>:<pk>`

- `DaoCachePrefix`：组件级前缀，默认 `daocache`。
- `DaoCachePackagePrefix`：服务级命名空间，默认空；多个服务共用 Redis 时用于隔离。
- 空段会被跳过，因此升级到带前缀版本不会导致既有缓存整体失效。
- `GetUniqKey()` 返回 `db:table:pk`，是业务唯一标识，**不含**缓存前缀。

```go
daoongorm.SetDaoCachePackagePrefix("identity_hub")
// key: identity_hub:daocache:identity_hub_test:user_groups:1
```

**负缓存**：`OptionNewForceCache(true)` 会把“查不到的行”也写入缓存，默认关闭；开启后需保证新增行能及时失效。

## 时间列精度

`created_at` / `updated_at` 自动填充的是纳秒精度的 `time.Now()`。若建表使用 `datetime(3)`，
直接落库会被截断/舍入，导致内存值与库中值不一致。按列声明 gorm 原生 `precision` 即可在填充时先行归一化：

```go
CreatedAt time.Time `gorm:"column:created_at;precision:3"` // datetime(3)
UpdatedAt time.Time `gorm:"column:updated_at;precision:6"` // datetime(6)
```

不声明 `precision` 时不做任何处理，值原样交给 gorm/driver（与旧版本行为一致）。

## 配置项

[option.go](./option.go) 提供 `NewDaoBase` 级别的选项：

| Option | 说明 |
|---|---|
| `OptionSetUseCache` | 是否启用缓存，默认 `true`（需配合 `CacheInterface`） |
| `OptionNewForceCache` | 记录不存在时是否也缓存，默认 `false` |
| `OptionSetCacheUtil` | 覆盖该 DAO 使用的缓存实现 |
| `OptionSetCacheExpireTS` | 覆盖缓存 TTL（秒），默认 7 天 |
| `OptionSetFieldNameCreatedAt` | 自定义 `created_at` 字段名，默认 `CreatedAt` |
| `OptionSetFieldNameUpdatedAt` | 自定义 `updated_at` 字段名，默认 `UpdatedAt` |

包级全局配置：

```go
daoongorm.DaoCacheExpire = 86400 * 7      // 默认 TTL
daoongorm.SetGlobalCacheUtil(myCache)     // 全局缓存实现
daoongorm.SetDaoCachePackagePrefix("app") // 缓存 key 命名空间
```

`DBClient` 支持主从 DSN、连接池、超时与日志级别：

```yaml
mysql:
  test:
    dsn: { userName: 'user', password: 'pass', endpoint: { address: '127.0.0.1', port: 3306 }, dbName: 'test' }
    readDSN: []          # 只读副本，空则复用主库
    active: 200
    idle: 2
    idleTimeout: '4h'
    queryTimeout: '1000ms' # → driver readTimeout
    execTimeout: '5000ms'  # → driver writeTimeout
    tranTimeout: '5000ms'  # → 事务超时
    logLevel: 'warn'       # silent / error / warn / info
```

## 示例与测试

```bash
# 单元测试：不依赖任何外部服务
go build ./... && go vet ./... && go test .

# 集成测试：需要真实 MySQL + redis
go test ./examples/ -v
```

[examples/](./examples) 是完整的集成示例：

- [examples/dao_test.go](./examples/dao_test.go)：CRUD、缓存、事务
- [examples/cache_util_demo.go](./examples/cache_util_demo.go)：基于 redigo 的 `CacheInterface` 实现
- [examples/config_demo.yaml](./examples/config_demo.yaml)：MySQL / redis 连接配置
- [examples/sql.md](./examples/sql.md)：测试表结构

运行集成测试前请导入建表 SQL，并按实际环境修改 `examples/config_demo.yaml`（默认连本地 3306）。

## 文档

- [AGENTS.md](./AGENTS.md) — 协作约定、兼容性红线、本地验证方式
- [harness/coding_style.md](./harness/coding_style.md) — 代码约束
- [harness/knowledge/architecture.md](./harness/knowledge/architecture.md) — 分层与调用链路
- [harness/knowledge/data_model.md](./harness/knowledge/data_model.md) — 模型与注册规则
- [harness/knowledge/crud.md](./harness/knowledge/crud.md) — CRUD 精确语义
- [harness/knowledge/cache.md](./harness/knowledge/cache.md) — 缓存策略与 Key 结构
- [harness/knowledge/db_client.md](./harness/knowledge/db_client.md) — 连接、事务、超时
- [harness/knowledge/testing.md](./harness/knowledge/testing.md) — 测试与本地容器

## 兼容性

本库被多个业务服务引用，**向后兼容优先**。以下行为属于对外契约，变更前需评估影响：

- `Save` 在主键为空时走 `Create`，否则走 `Update`。
- 未显式修改 `updated_at` 时，`Save` 会将其 touch 到当前时间。
- 只读 DAO 上的写操作返回错误，不静默降级。
- 默认缓存 key 格式 `daocache:<db>:<table>:<pk>` 保持不变。
- 事务内不写缓存，缓存失效只在 `COMMIT` 之后执行。

## License

[MIT](./LICENSE)
