# Architecture

一个 DAO = 一个 struct = 一行数据。`DaoBase` 把“按主键读写一行 + 缓存 + 时间戳”收在一处，
调用方不需要写 where 条件和缓存代码。

## 四层与各自边界

| 层 | 代表 | 职责 | 不做什么 |
|---|---|---|---|
| 模型 | `Model` 接口 + DAO struct | 声明表名/库名、列映射、主键 | 不含 SQL、不含缓存逻辑 |
| 元数据 | `ModelDef` / `Field` / `FieldTag` | 注册期反射解析并校验 | 不做 IO |
| DAO | `DaoBase` | 单行 CRUD、时间填充、缓存读写与失效 | 不做跨表逻辑、不开事务 |
| 连接 | `DBClientInterface` / `DBClient` | 读写分离、事务、超时、DSN | 不关心业务 |

缓存通过 `CacheInterface` 注入，默认 `NopCacheUtil`（不缓存）。

## Model 接口

```go
type Model interface {
	TableName() string
	DBName() string
	DBClient() DBClientInterface
	GetDaoBase() *DaoBase
	SetDaoBase(myDaoBase *DaoBase)
}
```

DAO struct 里放一个私有 `daoBase *DaoBase`，并提供 `NewXxxDao(ctx, dao, options...)`
把构造出来的 base 塞回去。这个 `NewXxxDao` 是给业务用的入口，examples 里的四个 DAO 就是模板。

## 注册

`RegisterModel(m)` 必须在 `init()` 里调用，把 struct 反射成 `ModelDef` 存进进程级
`modelDefCache`（`model_def_cache.go`）。`NewDaoBase*` 用 `pkg.Type` 全名查表，查不到直接返回
`模型未注册, <fullName>`。

注册期会 panic 的情形见 `data_model.md`：没有主键、列名重复、字段类型不支持、传入的不是 *struct。
这些都是**写错代码**，不是运行时输入，所以用 panic 尽早暴露。

## 一次 NewDaoBase 的完整链路

```
NewDaoBase(ctx, &User{ID: 1}, OptionSetReadOnly(true), OptionSetTX(tx))
  └─ GetModelFullNameAndValue      校验 *struct，拿可写 reflect.Value
  └─ modelDefCache 查 ModelDef      未注册 → error
  └─ applyOptionDefaults           tx=nil、isReadOnly=true(默认只读), 填缓存/TTL/时间字段名默认值
  └─ 应用 options                  OptionSetReadOnly / OptionSetTX 在这里生效
  └─ Load(ctx)
       ├─ 主键为空 → setFieldDefaultValue，标记 isNewRow
       ├─ 只读 + 启用缓存 → getFromCache 命中即返回（isLoadFromDB=false）
       ├─ 否则 SELECT ... LIMIT 1
       │    · 只读 → ReadOnlyTable（读副本）
       │    · 事务 → Table + FOR UPDATE
       ├─ RowsAffected<1 → isNewRow=true
       └─ 非事务且命中/存在 → setCache（含 newForceCache 的负缓存）
  └─ oldDataMap = 当前列快照      供 Update 计算差量
```

## 关键设计取舍（不要“简化”）

- **构造即加载**：`NewDaoBase` 一定 `Load`，所以 `IsNewRow()` 在构造后立刻可用。
  只读一次读取是 DAO 的主要用法，拆成两步只会让调用方忘记 Load。
- **Snapshot 用列名**：`oldDataMap` 以**列名**为 key（`GetColumnMapView`），不是 `structs` tag。
  tag 和列名不一致时（历史 DAO 就有），按 tag 取会永远判为“有变化”。
- **事务借 `AddAfterCommit`**：`DaoBase` 只依赖 `DBClientInterface`（两个方法）。要在提交后失效缓存，
  用可选接口 `afterCommitRegistry` 探测，而不是把方法塞进 `DBClientInterface`——外部自定义实现
  不应该被迫实现新方法。未实现时退化为“事务内不写缓存”。
- **读写分离在 `DBClient` 内部**：DAO 只说“我要只读/可写”，副本轮询、事务 clone 都在连接层。
