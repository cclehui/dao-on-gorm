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

签名：`Update(ctx context.Context, options ...UpdateOption) error`。

1. **前置校验**（全部在 `DaoBase.checkUpdatable()`，业务层不要再抄一遍），依次：
   `readOnly` → `load the row before Update` → `primary key ID is empty` → `row not found`（Load 过但库里没这行）
   → tx 与 Load 时不一致。
2. `updateAutoFillValue`：调用方**没有**改过 `updated_at` 时，把它 touch 成当前时间。
3. `roundTimeFields`。
4. `changedColumnValues`：与 `oldDataMap` 比较，**只 UPDATE 有变化的非主键列**。
5. 刷新 `oldDataMap`，更新缓存（事务内延迟到提交后）。

`Update` 专属的 Option 只有 `UpdateOptionSetTX(tx)`。`UpdateOption` 的接口方法不导出，外部包实现不了，
以后也不会往里加只读/缓存这类"Load 阶段就已经定死"的配置——在已经 Load 好的 DAO 上改它们，
调用方会以为生效了，其实没有。

- 校验失败时会把 `tx` 还原成调用前的值：被拒的 `Update` 不能把 DAO 留在"绑着别人事务"的状态。
- 不判 `isLoadFromDB`：从缓存 Load 只发生在只读模式（已被 `readOnly` 拦掉）；而
  `Create` 之后 `isLoadFromDB` 仍是 false，判它会把"创建后再改再存"误伤。
- **tx 一致性**：`Load` 时记下 `loadTx`，`Update` 传进来的 tx 必须是同一个。只有 UPDATE 进事务、
  Load 在事务外时，基线（`oldDataMap`）来自另一条连接、也没拿到 `FOR UPDATE` 锁，
  并发下两个事务会各读旧值互相覆盖。

注意第 2 步会先把 `updated_at` touch 到当前时间，所以“业务字段没改”的 `UpSert` 仍会发出一条
**只更新 `updated_at`** 的 UPDATE——这是原库的约定（`UpSert` 即 touch），examples 也断言了这点。
只有第 2 步也被跳过（例如字段名配错、根本没有 `updated_at` 字段）且其他列都没变时，才会提前返回不发 SQL。
如果调用方自己显式改了 `updated_at`，第 2 步不会覆盖它。

部分更新的意义：全列更新会把并发读到的其他字段旧值一起写回，A 和 B 同时改不同列时会互相覆盖。

比较规则：`time.Time` 用 `Equal`，其余用 `reflect.DeepEqual`。不要换回 `==` 或只看指针。

## UpSert

`IsNewRow() ? Create : Update`。这是唯一的“自动识别”，不要在其他地方再猜一次。

签名：`UpSert(ctx context.Context, options ...UpdateOption) error`。`options` 只在 `Update` 分支生效
（`Create` 分支没有基线、也没有"换 tx 写入"这回事）；`Save` 已改名，旧名不再保留。

## Delete

- 必须有主键，否则 `primary key ID is empty`。
- 按主键 `WHERE` 删除（不是把整个 model 交给 gorm）。
- 删除缓存：事务外立即删，事务内延迟到提交后。
- **物理删除**，本库没有软删概念；需要软删就在业务侧用状态列 + 自定义 selector。

## 事务

```go
err := dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
	row, err := NewUserDao(ctx, &User{ID: 1}, // 注意：必须带 OptionSetTX
		daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
	if err != nil {
		return err
	}
	row.Name = "x"
	return row.GetDaoBase().UpSert(ctx, daoongorm.UpdateOptionSetTX(tx))
})
```

- `Load` 用 `OptionSetTX(tx)`，`Update` / `UpSert` 用 `UpdateOptionSetTX(tx)`：两者必须是同一个 tx，
  否则报错。tx 在 `Load` 时就已绑定，`UpSert` 这里显式再传一次只是让"这条写落在哪个事务上"在调用点可见。
- 派生能力：`UpSert` 主键为空走 `Create`，此时 `options` 用不上（没有基线可写，事务由构造时的
  OptionSetTX 决定），传了也会被忽略。
- `Transaction` 用 `context.WithTimeout(ctx, conf.TranTimeout)` 开事务，出错或 panic 回滚。
- 回调里的 `tx` 是 clone：只有一个写连接，`ReadOnlyTable` 也落在同一个事务上。
- **事务里漏传 `OptionSetTX(tx)` 会走主库非事务连接**，不受回滚保护。这是最容易犯的错。
- 缓存失效在 COMMIT 成功之后统一执行（`AddAfterCommit`）；回滚时缓存保持原值。

## 只读 DAO

`NewDaoBaseNoLoad(model)` 给“已经用原生 SQL 查出数据”的场景：拿到 daoBase 后可以
`UpSert`/`Delete`/`CacheKey`，但它的 `isReadOnly = true`，写操作会返回 `readOnly`。
需要写就用 `NewDaoBase`。

## 写操作（service）

示例见 `examples/cclehui_test_a_service.go`：selector 只读，**写操作放 service**，负责参数组装与
业务校验，落库仍走 `DaoBase`。

核心约定：**先可写 Load → 改字段 → `UpSert`**。service 只做校验，不搬运字段。
`Update` 的基线是 `Load` 那一刻的库值，所以只改想改的字段就够，没改的列不会进 UPDATE，
零值也能表达（"把 version 改成 0"）。

代价是调用方必须先 Load，没 Load 就报错——这是刻意的：

- 没 Load 时 `IsNewRow()` 默认是 true，`UpSert` 会走 `Create`，"更新不存在的数据"变成插入；
- 用 `&Dao{ID: 1, Name: "x"}` 这种只填几列的结构体，没填的 `Age` 是零值 `time.Time`，
  MySQL 直接报 `1292 Incorrect datetime value: '0000-00-00'`；
- **按主键读一行就用 `NewXxxDao`，不要另写 selector 方法**：构造即 Load，缓存与读副本
  都由构造时的 Option 决定。默认只读，写不进去；更新要传
  `OptionSetReadOnly(false)` 做可写 Load（基线来自主库，而不是可能过期的副本）。

Create 时在 service 里把复杂参数编码成列值。示例把编解码方法放在类型自己身上：
`(*CclehuiTestAExtra).ToJSON()` / `(*CclehuiTestADao).GetExtra()`，service 只负责传结构体，
调用方不用关心 `column_default:{}` 这类存储细节。`ToJSON` 的 nil 接收者返回 `{}`。

### 事务透传

`Create` / `Update` / `Delete` 都收 `options ...daoongorm.Option`，上层事务直接透传：

```go
err := dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
	_, err := service.Create(ctx, params, daoongorm.OptionSetTX(tx))
	return err
})
```

- `OptionSetTX(tx)` 只能**透传**，不要在 service 里自己开事务（嵌套事务不是本库的语义）。
- `Create` / `Delete` 把 `options` 交给 `NewXxxDao`，并在最后补一个
  `OptionSetReadOnly(false)` 覆盖调用方的值——这两个 DAO 是 service 自己造的。
- `Update` 的 DAO 是调用方造的，**不补** `OptionSetReadOnly(false)`：把调用方的只读 Load
  悄悄改成可写，等于绕过"只读 DAO 的写操作必须报错"。调用方要写就自己传
  `OptionSetReadOnly(false)` 做可写 Load。
- `Update` 的 Option 类型是 `UpdateOption`（只有 `UpdateOptionSetTX`），`Create` / `Delete` 是
  通用的 `Option`。这不是笔误：构造函数能接受只读/缓存配置，而已经 Load 好的 DAO 只该换 tx。
- **业务层不重复校验**：`nil`、主键、只读、未 Load、行不存在、tx 不一致全在
  `DaoBase.checkUpdatable()` 里，service 只做 `nil` 入参兜底 + 透传 + 包一层上下文错误：

  ```go
  row, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: id},
      daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
  // 改字段...
  _, err = service.Update(ctx, row, daoongorm.UpdateOptionSetTX(tx))
  ```

事务里用只读方式（默认）读**看不到自己未提交的写**：只读走读副本、非事务连接。
需要确认事务内的写入就带同一个 tx 做可写 Load。

## 列表查询（selector）

单行 DAO 只负责按主键读写一行（用 `NewXxxDao` 构造即 Load）；**列表/条件查询不属于 DaoBase**，
写在 selector 里，直接用
`ReadOnlyTable(...).Select("*").Where(...).Find(&list)`。

- 分页用游标（`id > ?` + `ORDER BY id`），不要用 `OFFSET`：翻页期间删行会漏数据。
- `Find` 出来的结构体没有 `daoBase`，要补 `NewDaoBaseNoLoad`，否则调用方拿到的对象不能
  用统一的只读视图（写操作也不会报 `readOnly`）。
- 参数收成一个 `XxxParams` struct：新增条件时不用改方法签名，调用方也看得懂每个字段含义。
  查询相关的辅助逻辑（默认 Limit、排序白名单、过滤条件）写成 **selector 的私有方法**
  （`func (selector *XxxSelector) orderColumn(params *XxxParams)`），放在公开查询方法之后，
  不要散成包级变量/常量或包级函数。
- 列表和它的总数共用一个 `XxxParams`，过滤条件走同一个 `applyFilters`，保证两边口径一致；
  `Limit` / `OrderBy` 只影响 list，不影响 count。
- `ORDER BY` 的列名是标识符，`?` 绑定不了，必须走**白名单**（`orderColumn()` 里的 switch），
  把调用方的字符串直接拼进 SQL 就是注入。
- `LIKE` 的用户输入必须转义 `%` / `_` / `\`，否则 `a_b` 会匹配 `axb`，`%` 会变成"匹配所有"。
  统一用 `utils.SQLUtil.LikeContains(...)` + `escape '\'`，不要在 selector 里自己拼。
  注意这是**匹配语义**问题；注入防护靠绑定参数，见 [sql_injection.md](./sql_injection.md)。

