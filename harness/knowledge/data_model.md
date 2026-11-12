# Data Model

## DAO 结构体模板

```go
type CclehuiTestADao struct {
	ID        int       `gorm:"column:id;primaryKey" structs:"id" json:"id"`
	Version   int64     `gorm:"column:version" structs:"version" json:"version"`
	Weight    float64   `gorm:"column:weight;column_default:1.9" structs:"weight" json:"weight"`
	Age       time.Time `gorm:"column:age" structs:"age" json:"age"`
	Extra     string    `gorm:"column:extra" structs:"extra" json:"extra"`
	CreatedAt time.Time `gorm:"column:created_at;precision:3" structs:"created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;precision:3" structs:"updated_at" json:"updated_at"`

	daoBase *daoongorm.DaoBase
}

func init() { daoongorm.RegisterModel(&CclehuiTestADao{}) }
```

- struct 名/字段名不用和表/列一致，但必须通过 `gorm:"column:..."` 显式声明列名。
  没写 `column:` 时按字段名转 snake_case，容易和实际表结构对不上。
- 主键必须有 `primaryKey`，联合主键在多个字段上各标一次。
- 私有 `daoBase` 字段不会被映射（未导出字段直接跳过）。

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

`NewDaoBase` 和 `NewDaoBaseWithTX` 都要带上同样的 Option，examples 里用
`options = append(options, ...)` 做在两处。
