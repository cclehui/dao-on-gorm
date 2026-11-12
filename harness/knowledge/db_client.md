# DBClient

`DBClient` 是 `DBClientInterface` 的默认实现：

```go
type DBClientInterface interface {
	Table(ctx context.Context, name string) *gorm.DB
	ReadOnlyTable(ctx context.Context, name string) *gorm.DB
}
```

DAO 只依赖这两个方法，连接池、读写分离、超时都在实现内部。
自定义实现（比如接入公司统一 DB SDK）只要能给出这两个方法即可，`DaoBase` 不需要改。

## 读写分离

- `Table` → 主库（写连接）。
- `ReadOnlyTable` → `ReadDSN` 轮询（原子索引），未配置副本时回落到主库。
- 事务 clone 的 `read` 就是事务本身，所以事务里 `ReadOnlyTable` 不会跑到副本。

## ctx

`DataSource` 统一用 `.WithContext(ctx)`。历史上这里是
`Set("scope_context", ctx)`，ctx 根本没到 gorm，请求取消和 trace 全部失效——不要改回去。

## 事务

```go
dbClient.Transaction(ctx, func(ctx context.Context, tx *DBClient) error { ... })
```

- 用 `context.WithTimeout(ctx, conf.TranTimeout)`，超时会随 SQL 一起取消。
- 回调返回 error 或 panic → 回滚；正常返回 → 提交。
- 提交成功后才执行 `AddAfterCommit` 注册的回调（缓存失效）。
- 回调参数里的 `ctx` 带事务 deadline（`TranTimeout`），事务结束后即失效，不要存到事务外继续用。
- 提交后的缓存失效回调拿到的是**外层** `ctx`（`Transaction` 的入参），不是带事务 deadline 的那个。

## 配置项

```go
type DBClientConfig struct {
	DSN        *DSNConfig
	ReadDSN    []*DSNConfig
	Active     int           // max open conns
	Idle       int           // max idle conns
	IdleTimeout ctime.Duration // conn max lifetime
	QueryTimeout ctime.Duration // → driver readTimeout
	ExecTimeout  ctime.Duration // → driver writeTimeout
	TranTimeout  ctime.Duration // 事务超时
	LogLevel     string         // silent / error / warn / info，默认 warn
}
```

- `QueryTimeout` / `ExecTimeout` 会拼进 DSN 的 `readTimeout` / `writeTimeout`；
  调用方在 `Options` 里显式写了同名字段时**以显式值为准**，不会被覆盖。
- `LogLevel` 空或未知值 → `warn`。生产不要用 `info`，每条 SQL 一行日志。
- `Active` / `Idle` / `IdleTimeout` 为 0 时不设置，交给 `database/sql` 默认值。

## DSN

`concatConnectURI` 拼 `user:pass@(host:port)/db?opts`。`Options` 原样追加，
所以 `charset=utf8mb4&parseTime=true&loc=UTC` 这类由使用方负责。

时间相关建议：`parseTime=true` 必开，否则 `time.Time` 字段读不回来；
`loc` 决定读回时的时区，跨服务统一成 `UTC` 最省事。
