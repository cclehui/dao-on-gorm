# dao-on-gorm

English | [简体中文](./README.md)

[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

`dao-on-gorm` is a **single-row DAO** layer built on top of [gorm](https://gorm.io/):

> One struct maps to one row. `DaoBase` handles `Load / Create / Update / Save / Delete` by primary
> key, and takes care of primary-key caching plus automatic `created_at` / `updated_at` filling.

Callers no longer hand-write `WHERE` clauses or scatter cache logic across business code.

```go
user, err := NewUserDao(ctx, &User{ID: 1}, false)
if err != nil {
    return err
}

user.Name = "connor"
return user.GetDaoBase().Save(ctx) // automatically picks Insert or Update
```

## Features

- **Single-row semantics**: one DAO = one row, addressed by primary key. No list or conditional queries.
- **Load on construction**: `NewDaoBase` loads automatically, so `IsNewRow()` is usable right away.
- **Automatic timestamps**: zero-value `created_at` / `updated_at` are filled in, with per-column precision normalization.
- **Partial updates**: `Save` only writes columns that actually changed, avoiding lost updates under concurrency.
- **Primary-key cache**: single-row reads check the cache first, writes invalidate it. Caching is off by default and pluggable through `CacheInterface`.
- **Transaction safety**: rows are locked with `FOR UPDATE` inside transactions, and cache invalidation is deferred until after `COMMIT`.
- **Read/write splitting**: `DBClient` handles read-replica round-robin and transaction cloning transparently.
- **Extensible**: both `CacheInterface` and `DBClientInterface` can be implemented by the caller.

## Installation

```bash
go get github.com/cclehui/dao-on-gorm
```

Requires Go `1.21+`, `gorm.io/gorm v1.31.2`, and `gorm.io/driver/mysql v1.6.0`.

## Quick Start

### 1. Define the model

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

### 2. Register the model

Models must be registered during process startup: `NewDaoBase` relies on metadata parsed at registration time.

```go
func init() {
    daoongorm.RegisterModel(&User{})
}
```

An unregistered model returns `模型未注册, <type>` ("model not registered"). Coding errors such as a
missing primary key or duplicate column names panic at registration time.

### 3. Provide constructors

```go
// One constructor; read-only and transactions are expressed as options:
//   NewUserDao(ctx, user)                                        // read-only (default)
//   NewUserDao(ctx, user, daoongorm.OptionSetReadOnly(false))    // writable
//   NewUserDao(ctx, user, daoongorm.OptionSetReadOnly(false),
//       daoongorm.OptionSetTX(tx))                               // writable inside a transaction
func NewUserDao(ctx context.Context, user *User, options ...daoongorm.Option) (*User, error) {
    base, err := daoongorm.NewDaoBase(ctx, user, options...)
    user.daoBase = base
    return user, err
}
```

Model-specific options (for example custom timestamp field names) are appended inside the constructor.

Read-only mode and transactions are both options: `OptionSetReadOnly(bool)` / `OptionSetTX(tx)`.
`NewDaoBase` is the only construction entry point and is **read-only by default** — pass
`OptionSetReadOnly(false)` when you need to write.

`NewDaoBase` loads immediately during construction:

| Input | Behavior |
|---|---|
| Empty primary key | No query; fields default from `column_default`; `IsNewRow() == true` |
| Primary key set, `readOnly=true` (default) | Cache first, then read replica on miss |
| Primary key set, `readOnly=false` | Reads the primary; uses `FOR UPDATE` inside a transaction |

### 4. CRUD

```go
ctx := context.Background()

// Create: empty primary key means Save performs an Insert
user, err := NewUserDao(ctx, &User{Name: "connor"}, false)
if err != nil {
    return err
}
err = user.GetDaoBase().Save(ctx)
// user.ID is populated and IsNewRow() is now false

// Load: read by primary key (readOnly=true uses cache + read replica)
user, err = NewUserDao(ctx, &User{ID: 1}, true)

// Update: only changed columns are written; updated_at is touched automatically
user.Name = "new name"
err = user.GetDaoBase().Save(ctx)

// Delete: physical delete by primary key, then cache invalidation
err = user.GetDaoBase().Delete(ctx)
```

> If `updated_at` was not modified explicitly, `Save` touches it to the current time. A `Save` with no
> business-field changes therefore still emits an `UPDATE` that only sets `updated_at`. This is an
> intentional, long-standing contract of the library.

### 5. Transactions

```go
err := dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
    user, err := NewUserDao(ctx, &User{ID: 1},
        daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
    if err != nil {
        return err
    }

    user.Name = "connor"
    return user.GetDaoBase().Save(ctx)
})
```

- Inside a transaction you **must** pass `OptionSetTX(tx)`. Otherwise the DAO uses
  the primary non-transactional connection and the work is not protected by rollback.
- Single-row reads inside a transaction acquire a `FOR UPDATE` row lock.
- Cache invalidation is registered via `AddAfterCommit` and runs only after a successful `COMMIT`;
  on rollback the existing cache entries stay untouched.

## Cache

The cache stores the **entire row struct** keyed by primary key, and only affects single-row reads by
primary key. List and conditional queries bypass it. The default implementation is `NopCacheUtil`
(no caching).

```go
type CacheInterface interface {
    Set(ctx context.Context, key string, value interface{}, ttl int) error
    Get(ctx context.Context, key string, value interface{}) (hit bool, error)
    Del(ctx context.Context, key string) error
}
```

Register globally or override per DAO:

```go
daoongorm.SetGlobalCacheUtil(myCache)

user, err := NewUserDao(ctx, &User{ID: 1}, true,
    daoongorm.OptionSetCacheUtil(myCache),
    daoongorm.OptionSetCacheExpireTS(3600),
)
```

**Key format**: `[package_prefix:]daocache:<db>:<table>:<pk>`

- `DaoCachePrefix`: component-level prefix, defaults to `daocache`.
- `DaoCachePackagePrefix`: service-level namespace, empty by default. Useful when several services share
  one Redis instance.
- Empty segments are skipped, so upgrading to a prefixed version does not invalidate existing cache entries.
- `GetUniqKey()` returns `db:table:pk`, the business identity, and **does not** include cache prefixes.

```go
daoongorm.SetDaoCachePackagePrefix("identity_hub")
// key: identity_hub:daocache:identity_hub_test:user_groups:1
```

**Negative caching**: `OptionNewForceCache(true)` also caches rows that were not found. It is off by
default; when enabled, make sure newly inserted rows invalidate the corresponding entry promptly.

## Time column precision

`created_at` / `updated_at` are filled with a nanosecond-precision `time.Now()`. If the column is
`datetime(3)`, MySQL truncates or rounds the value on write, leaving the in-memory value out of sync with
the stored one. Declare gorm's native `precision` to normalize before the value reaches gorm/driver:

```go
CreatedAt time.Time `gorm:"column:created_at;precision:3"` // datetime(3)
UpdatedAt time.Time `gorm:"column:updated_at;precision:6"` // datetime(6)
```

Without `precision` the value is passed through unchanged, matching the legacy behavior. Precision is a
per-column attribute, so different tables can use different values.

## Options

[option.go](./option.go) defines per-DAO options:

| Option | Description |
|---|---|
| `OptionSetUseCache` | Enable or disable caching. Defaults to `true` (requires a `CacheInterface`) |
| `OptionNewForceCache` | Cache rows that were not found. Defaults to `false` |
| `OptionSetCacheUtil` | Override the cache implementation for this DAO |
| `OptionSetCacheExpireTS` | Override the cache TTL in seconds. Defaults to 7 days |
| `OptionSetFieldNameCreatedAt` | Custom `created_at` field name. Defaults to `CreatedAt` |
| `OptionSetFieldNameUpdatedAt` | Custom `updated_at` field name. Defaults to `UpdatedAt` |

Package-level configuration:

```go
daoongorm.DaoCacheExpire = 86400 * 7      // default TTL
daoongorm.SetGlobalCacheUtil(myCache)     // global cache implementation
daoongorm.SetDaoCachePackagePrefix("app") // cache key namespace
```

`DBClient` supports primary/replica DSNs, connection pooling, timeouts, and log levels:

```yaml
mysql:
  test:
    dsn: { userName: 'user', password: 'pass', endpoint: { address: '127.0.0.1', port: 3306 }, dbName: 'test' }
    readDSN: []          # read replicas; empty means reuse the primary
    active: 200
    idle: 2
    idleTimeout: '4h'
    queryTimeout: '1000ms' # → driver readTimeout
    execTimeout: '5000ms'  # → driver writeTimeout
    tranTimeout: '5000ms'  # → transaction timeout
    logLevel: 'warn'       # silent / error / warn / info
```

## Examples and testing

```bash
# Unit tests: no external services required
go build ./... && go vet ./... && go test .

# Integration tests: require a real MySQL + Redis (examples is a nested module)
cd examples && go test ./... -v
```

The [examples/](./examples) directory contains a complete integration example and is a **separate Go
module** (`examples/go.mod`, wired to the repository root via `replace`). Example-only dependencies such
as Redis, testify, and yaml therefore stay out of the root `go.mod` and are never pulled in by consumers
of this library.

- [examples/dao_test.go](./examples/dao_test.go): CRUD, caching, transactions
- [examples/model/](./examples/model): DAO structs (A/B/C/D covering the primary-key and timestamp
  variations) and model registration
- [examples/cclehui_test_a_selector.go](./examples/cclehui_test_a_selector.go): read queries (single
  row, count, filtered list with cursor paging)
- [examples/cclehui_test_a_service.go](./examples/cclehui_test_a_service.go): writes (Create / Update /
  Delete)
- [examples/global/](./examples/global): config loading, lazy `DBClient` / cache setup, and a
  redigo-based `CacheInterface`
- [examples/config_demo.yaml](./examples/config_demo.yaml): MySQL / Redis connection settings
- [examples/sql.md](./examples/sql.md): test table schemas

Import the schema SQL and adjust `examples/config_demo.yaml` for your environment before running the
integration tests (it defaults to local port 3306).

## Documentation

- [AGENTS.md](./AGENTS.md): collaboration rules, compatibility constraints, local verification
- [harness/coding_style.md](./harness/coding_style.md): coding conventions
- [harness/knowledge/architecture.md](./harness/knowledge/architecture.md): layers and call flow
- [harness/knowledge/data_model.md](./harness/knowledge/data_model.md): models and registration
- [harness/knowledge/crud.md](./harness/knowledge/crud.md): exact CRUD semantics
- [harness/knowledge/cache.md](./harness/knowledge/cache.md): caching strategy and key format
- [harness/knowledge/db_client.md](./harness/knowledge/db_client.md): connections, transactions, timeouts
- [harness/knowledge/testing.md](./harness/knowledge/testing.md): tests and local containers

## Compatibility

This library is consumed by several services, so **backward compatibility comes first**. The following
behaviors are part of the public contract and must be evaluated carefully before changing:

- `Save` performs `Create` when the primary key is empty, and `Update` otherwise.
- Unless `updated_at` was modified explicitly, `Save` touches it to the current time.
- Write operations on a read-only DAO return an error instead of silently degrading.
- The default cache key format `daocache:<db>:<table>:<pk>` stays stable.
- Nothing is written to the cache inside a transaction; invalidation only happens after `COMMIT`.

## License

[MIT](./LICENSE)
