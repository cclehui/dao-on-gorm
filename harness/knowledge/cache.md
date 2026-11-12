# Cache

按主键缓存**整行结构体**，只在“按主键读单行”这条路径上生效。列表/条件查询不走这里的缓存。

## Key 结构

```
[package_prefix:]daocache:<db>:<table>:<pk>
```

- `DaoCachePrefix`（`constant.go`，默认 `"daocache"`）：组件级前缀。
- `DaoCachePackagePrefix`（`constant.go`，默认 `""`）：包/服务级命名空间，用
  `SetDaoCachePackagePrefix("identity_hub")` 设置（会 TrimSpace）。
- `<pk>` 由 `PKFieldsStr()` 生成：联合主键按**字段名排序**后取值，用 `_` 连接。

空段会被跳过，所以默认配置下的 key 与老版本逐字节一致——升级不会让线上缓存整体失效。
`GetUniqKey()` 返回 `db:table:pk`，是业务唯一标识，**不含**任何缓存前缀。

## 什么情况下读写缓存

| 操作 | 事务外 | 事务内 |
|---|---|---|
| `Load`（只读）| 先读缓存，未命中回源后写入 | 不读也不写 |
| `Load`（可写）| 不读缓存 | 不读也不写 |
| `Create` | 写缓存 | 不写 |
| `Update` | 写缓存 | 不写，注册提交后失效 |
| `Delete` | 立即删 | 注册提交后删 |

事务内的规则只有一条：**未提交的数据不能进缓存，回滚不能丢缓存**。
实现是 `refreshCache` / `invalidateCache`：非事务直接操作，事务里通过
`DBClient.AddAfterCommit` 注册，COMMIT 成功后才执行。

## 负缓存

`OptionNewForceCache(true)` 让“查不到的行”也写入缓存（值为零值 model）。默认关闭。
开启后必须保证新增行能及时失效，否则会读到“不存在”的假象。

## CacheInterface

```go
type CacheInterface interface {
	Set(ctx context.Context, key string, value interface{}, ttl int) (err error)
	Get(ctx context.Context, key string, value interface{}) (hit bool, err error)
	Del(ctx context.Context, key string) (err error)
}
```

- 默认 `NopCacheUtil`，全部 no-op，等价于不缓存。
- 全局用 `SetGlobalCacheUtil(...)` 注入；单个 DAO 可用 `OptionSetCacheUtil(...)` 覆盖。
- 缓存错误只记日志，不影响主流程：`Get` 失败视为 miss，`Set/Del` 失败只告警。
- `value` 一定是 DAO struct 指针，实现方负责序列化。

## TTL

`DaoCacheExpire`（秒，默认 7 天）或 `OptionSetCacheExpireTS(...)`。
TTL 由缓存实现解释；`redis-util` 之类的实现按秒处理。
