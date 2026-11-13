# CRUD Semantics

精确行为，改动前先看 `AGENTS.md` 的兼容性红线。

## Load

`NewDaoBase` 构造时会自动 `Load(ctx)`；也可以手动调，重复调用是 no-op（`isLoaded` 标记）。

| 情况 | 行为 | 结果 |
|---|---|---|
| 主键为空 | 不查库，按 `column_default` 填默认值 | `IsNewRow() == true` |
| 只读 + 启用缓存 + 命中 | 直接从缓存反序列化到 model | `IsLoadFromDB() == false` |
| 只读 | `ReadOnlyTable`（读副本）`SELECT ... LIMIT 1` | 有行 → `IsNewRow()==false` |
| 可写 + 事务 | `Table(...)` + `FOR UPDATE` | 同上，行被锁 |
| 可写 + 非事务 | `Table(...)` `SELECT ... LIMIT 1` | 同上 |
| 查不到 | `RowsAffected < 1` | `IsNewRow() == true` |

注意：

- 用 `Scan` 而不是 `First`，所以“查不到”不是 error。
- 只读模式**只在请求时**查缓存；事务里不写缓存。
- 事务内的 `FOR UPDATE` 用 `clause.Locking`；gorm v2 不认 `Set("gorm:query_option")`。

## Create

1. `createAutoFillValue`：`created_at` / `updated_at` 为零值时填 `time.Now()`。
2. `roundTimeFields`：所有 `precision:` 字段按精度 `Round`。
3. `INSERT`。
4. 刷新 `oldDataMap`，事务外写缓存。

## Update

1. 必须 `isLoaded` 且有主键，否则报错（`需要先Load` / `primary key ID is empty`）。
2. `updateAutoFillValue`：调用方**没有**改过 `updated_at` 时，把它 touch 成当前时间。
3. `roundTimeFields`。
4. `changedColumnValues`：与 `oldDataMap` 比较，**只 UPDATE 有变化的非主键列**。
5. 刷新 `oldDataMap`，更新缓存（事务内延迟到提交后）。

注意第 2 步会先把 `updated_at` touch 到当前时间，所以“业务字段没改”的 `Save` 仍会发出一条
**只更新 `updated_at`** 的 UPDATE——这是原库的约定（`Save` 即 touch），examples 也断言了这点。
只有第 2 步也被跳过（例如字段名配错、根本没有 `updated_at` 字段）且其他列都没变时，才会提前返回不发 SQL。
如果调用方自己显式改了 `updated_at`，第 2 步不会覆盖它。

部分更新的意义：全列更新会把并发读到的其他字段旧值一起写回，A 和 B 同时改不同列时会互相覆盖。

比较规则：`time.Time` 用 `Equal`，其余用 `reflect.DeepEqual`。不要换回 `==` 或只看指针。

## Save

`IsNewRow() ? Create : Update`。这是唯一的“自动识别”，不要在其他地方再猜一次。

## Delete

- 必须有主键，否则 `primary key ID is empty`。
- 按主键 `WHERE` 删除（不是把整个 model 交给 gorm）。
- 删除缓存：事务外立即删，事务内延迟到提交后。
- **物理删除**，本库没有软删概念；需要软删就在业务侧用状态列 + 自定义 selector。

## 事务

```go
err := dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
	row, err := NewUserDaoWithTX(ctx, &User{ID: 1}, tx) // 注意：必须用 WithTX 构造
	if err != nil {
		return err
	}
	row.Name = "x"
	return row.GetDaoBase().Save(ctx)
})
```

- `Transaction` 用 `context.WithTimeout(ctx, conf.TranTimeout)` 开事务，出错或 panic 回滚。
- 回调里的 `tx` 是 clone：只有一个写连接，`ReadOnlyTable` 也落在同一个事务上。
- **事务里误用 `NewDaoBase` 会走主库非事务连接**，不受回滚保护。这是最容易犯的错，
  所有 `NewXxxDao` 都要同时提供 `NewXxxDaoWithTX`。
- 缓存失效在 COMMIT 成功之后统一执行（`AddAfterCommit`）；回滚时缓存保持原值。

## 只读 DAO

`NewDaoBaseNoLoad(model)` 给“已经用原生 SQL 查出数据”的场景：拿到 daoBase 后可以
`Save`/`Delete`/`CacheKey`，但它的 `isReadOnly = true`，写操作会返回 `readOnly`。
需要写就用 `NewDaoBase`。

## 写操作（service）

示例见 `examples/cclehui_test_a_service.go`：selector 只读，**写操作放 service**，负责参数组装与
业务校验，落库仍走 `DaoBase`。

核心约定：**先可写 Load → 改字段 → `Save`**。service 只做校验，不搬运字段。
`Update` 的基线是 `Load` 那一刻的库值，所以只改想改的字段就够，没改的列不会进 UPDATE，
零值也能表达（"把 version 改成 0"）。

代价是调用方必须先 Load，没 Load 就报错——这是刻意的：

- 没 Load 时 `IsNewRow()` 默认是 true，`Save` 会走 `Create`，"更新不存在的数据"变成插入；
- 用 `&Dao{ID: 1, Name: "x"}` 这种只填几列的结构体，没填的 `Age` 是零值 `time.Time`，
  MySQL 直接报 `1292 Incorrect datetime value: '0000-00-00'`；
- **不要用 selector 查出来的 DAO**：`GetDataByID` 走 `ReadOnlyTable` + `NewDaoBaseNoLoad`，
  它是只读的，写不进去，而且读的是副本、基线可能已经过期。更新要用
  `NewCclehuiTestADao(ctx, &Dao{ID: x}, false)` 做可写 Load。

Create 时在 service 里把复杂参数编码成列值。示例把编解码方法放在类型自己身上：
`(*CclehuiTestAExtra).ToJSON()` / `(*CclehuiTestADao).GetExtra()`，service 只负责传结构体，
调用方不用关心 `column_default:{}` 这类存储细节。`ToJSON` 的 nil 接收者返回 `{}`。

## 列表查询（selector）

单行 DAO 只负责按主键读写一行，**列表/条件查询不属于 DaoBase**，写在 selector 里，直接用
`ReadOnlyTable(...).Select("*").Where(...).Find(&list)`。

- 分页用游标（`id > ?` + `ORDER BY id`），不要用 `OFFSET`：翻页期间删行会漏数据。
- `Find` 出来的结构体没有 `daoBase`，要补 `NewDaoBaseNoLoad`，否则调用方拿到的对象不能
  用统一的只读视图（写操作也不会报 `readOnly`）。
- 参数收成一个 `XxxParams` struct：新增条件时不用改方法签名，调用方也看得懂每个字段含义。
- `LIKE` 的用户输入必须转义 `%` / `_` / `\`，否则 `a_b` 会匹配 `axb`，`%` 会变成"匹配所有"。
  统一用 `utils.SQLUtil.LikeContains(...)` + `escape '\'`，不要在 selector 里自己拼。
  注意这是**匹配语义**问题；注入防护靠绑定参数，见 [sql_injection.md](./sql_injection.md)。

