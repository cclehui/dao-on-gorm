# Data Model

## DAO 结构体模板

```go
type CclehuiTestADao struct {
	ID        int       `gorm:"column:id;primaryKey" structs:"id" json:"id"`
	Version   int64     `gorm:"column:version" structs:"version" json:"version"`
	Weight    float64   `gorm:"column:weight;column_default:1.9" structs:"weight" json:"weight"`
	Age       time.Time `gorm:"column:age" structs:"age" json:"age"`
	Name      string    `gorm:"column:name" structs:"name" json:"name"`
	Extra     string    `gorm:"column:extra;type:json;column_default:{}" structs:"extra" json:"extra"`
	CreatedAt time.Time `gorm:"column:created_at;precision:3" structs:"created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;precision:3" structs:"updated_at" json:"updated_at"`

	daoBase *daoongorm.DaoBase
}

// 注册紧跟在 struct 定义之后
func init() { daoongorm.RegisterModel(&CclehuiTestADao{}) }

func NewCclehuiTestADao(ctx context.Context, myDao *CclehuiTestADao,
	options ...daoongorm.Option) (*CclehuiTestADao, error) {
	daoBase, err := daoongorm.NewDaoBase(ctx, myDao, options...)

	myDao.daoBase = daoBase

	return myDao, err
}
```

- 单个 DAO 文件内的顺序固定为：`struct` → `init()`（`RegisterModel`）→ `NewXxxDao` →
  `DBName` / `TableName` / `DBClient` / `GetDaoBase` / `SetDaoBase` → 私有 helper。
  即"公开方法在前、私有 helper 在后"，通用约定见 `harness/coding_style.md` 的
  「结构内聚与函数顺序」。每个 DAO 在自己的文件里注册，不要再建集中的 `init.go`。
- struct 名/字段名不用和表/列一致，但必须通过 `gorm:"column:..."` 显式声明列名。
  没写 `column:` 时按字段名转 snake_case，容易和实际表结构对不上。
- 主键必须有 `primaryKey`，联合主键在多个字段上各标一次。
- 私有 `daoBase` 字段不会被映射（未导出字段直接跳过）。
- 示例 DAO 放在 `examples/model/` 独立 package 里，`Model.DBClient()` 直接调
  `global.GetDBClient()`。依赖方向是 `model -> global`，`global` **不要** import `model`，
  否则成环。

## 支持的 tag

`gorm` tag 由本库自己解析（`FieldTag`），**不是** gorm 的原生 schema：

| tag | 含义 |
|---|---|
| `column:<name>` | 列名 |
| `primaryKey` | 主键 |
| `column_default:<v>` / `default:<v>` | 主键为空、准备新建时填给该字段的默认值 |
| `precision:<n>` | 时间列的小数秒精度，见下 |
| `-` 或空 tag | 忽略该列 |

`precision` 同时会被 gorm 读取（用于 DDL），本库也读它（用于写入前的舍入）。

## JSON 列

本库只映射基础类型和 `time.Time`，**没有** `map` / struct / 自定义 JSON 类型的分支，所以 JSON 列
只能用 `string` 承载 JSON 文本：

```go
Extra string `gorm:"column:extra;type:json;column_default:{}" structs:"extra" json:"extra"`
```

两个必须注意的点（都用 MySQL 8 实测过）：

- **空串不是合法 JSON**。新建行时字段是零值 `""`，直接写进 `json` 列会报
  `Error 3140: Invalid JSON text: "The document is empty."`。必须给 `column_default:{}`
  （或让调用方显式赋合法 JSON）。
- **MySQL 会重写 JSON 文本**：键顺序按长度/字典重排、去掉多余空格。例如写入
  `{"b":2,"a":1}`，读回来是 `{"a": 1, "b": 2}`。`Update` 的“列是否变化”判定是按字符串
  比较的，所以写完立刻再 `Save` 一次会把 `extra` 多更新一遍——结果幂等，但会多一条写。
  要避免这次多余写入，就在 `Load` 之后以库里的值为基准比较。

## 时间列精度

Go 的 `time.Now()` 是纳秒精度，MySQL 的 `datetime(3)` 只存毫秒，并且在写入时**四舍五入**
（`.123600` → `.124000`），不是截断。

gorm 的 `precision:` **只影响 DDL**，不会修改你传进去的值（实测 gorm v1.31.2）。因此：

- 声明了 `precision:3` 的字段，`Create`/`Update` 前会 `Round` 到该精度，
  保证内存里的值 = 库里的值；
- 没有声明 `precision` 的字段**不做任何处理**，原样交给 gorm/driver；
- `precision:6` 用于 `datetime(6)`。

不这么做的后果：内存值与落库值永久不一致，`Update` 的“字段是否变化”判定会把
`updated_at` 当成每次都变了，还会让审计时间对不上。

## 字段类型限制（注册期校验）

只支持：`bool`、有/无符号整数、`float32/64`、`string`、`time.Time`。

`map`、`slice`、指针、自定义 struct 一律 panic。原因：反射赋值路径没有对应分支，
注册成功也只能静默写零值，比直接报错更难查。

## 注册期 panic 清单

- 传入的不是 `*struct`（包括 `nil` 指针、值类型）
- 没有任何列
- 没有主键
- 两个字段映射到同一列
- 字段类型不支持
- `column_default` 无法解析成字段类型

## 时间字段名

自动填充默认按**字段名** `CreatedAt` / `UpdatedAt` 查找，不是按列名。所以：

- 字段名仍是 `CreatedAt`、只是列名不同（`column:created_at_new`）→ 不需要任何 Option，
  `precision` 照常生效；
- 字段名也变了（examples 里的 DAO B 用 `CreatedAtNew` / `UpdatedAtNew`）→ 必须传
  `OptionSetFieldNameCreatedAt("CreatedAtNew")` / `OptionSetFieldNameUpdatedAt("UpdatedAtNew")`，
  否则这两个字段不会被自动填充。

`precision` 的舍入（`roundTimeFields`）是遍历所有声明了精度的字段，**不受字段名影响**：
即使忘了配 Option，只要写了 `precision:3`，显式赋的值仍会被舍入。

只读/事务也是 Option（`OptionSetReadOnly` / `OptionSetTX`），构造统一走 `NewDaoBase`，
且**默认只读**（`applyOptionDefaults` 里设为 true）。

`NewXxxDao` 只保留一个：`NewXxxDao(ctx, dao, options...)`，没有 `NewXxxDaoWithTX`。
调用方按需传 `OptionSetReadOnly(false)` / `OptionSetTX(tx)`；模型自己的固定 Option
（例如 DAO B 的自定义时间字段名）在构造函数里 `append` 进去。
