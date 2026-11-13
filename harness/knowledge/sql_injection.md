# SQL 注入防护

结论先行：**危险的是把值拼进 SQL 文本，不是 `?` 或 prepared statement 本身**。
参数化是手段，`?` 是 gorm 里表达参数化的写法，prepared statement 只是一种实现。

## 为什么 `?` 是安全的

`Where(" name = ? ", v)` 不会做字符串替换：gorm 把 `v` 放进 `Statement.Vars`，SQL 文本里
只留 `?`。值在协议层单独传输，SQL 解析器从头到尾没见过它。

MySQL 8 的 general log 实测（`Name: "' or '1'='1"`）：

```
Prepare  SELECT * FROM `cclehui_test_a` WHERE  name = ?  AND  id > ?  ORDER BY id asc LIMIT ?
Execute  SELECT * FROM `cclehui_test_a` WHERE  name = '\' or \'1\'=\'1'  AND  id > 0 ...
```

`Execute` 那行是 MySQL **展示**绑定值的方式，解析器只见过 `?`。

## 两种实现都没问题

| 模式 | 服务端看到 | 谁负责转义 |
|---|---|---|
| 默认（本库） | `Prepare` + `Execute`，SQL 里是 `?` | 不涉及，值走参数通道 |
| DSN 加 `interpolateParams=true` | `Query`，值是 client 拼好的字面量 | go-sql-driver 的引号转义函数 |

两种情况都实测过：`' or '1'='1`、`'; drop table cclehui_test_a; -- `、`a\'b` 传给 `Name`
和 `NameLike` 都返回 0 行、无报错、表还在。

不要因为 `interpolateParams` 下 SQL 文本里出现了值就以为"这也是拼接所以不安全"——
那是驱动在按 MySQL 字面量规则转义，和你手动 `+` 拼接不是一回事。

## 反例（会注入）

```go
db.Where("name = '" + v + "'")            // 引号没人处理
db.Where(fmt.Sprintf("name = '%s'", v))   // 同上
db.Order(v)                               // 标识符不能参数化
```

## `?` 覆盖不到的地方

值可以参数化，**标识符和 SQL 结构不行**：列名、表名、`ORDER BY` 的列与方向、`LIMIT` 之外
的语法片段。这些一旦来自用户输入，就只能靠**白名单**：

```go
var allowedSort = map[string]bool{"created_at": true, "id": true}
if !allowedSort[params.SortBy] { params.SortBy = "id" }
```

`LIMIT` 用 `Limit(n)` / `?` 绑定；不要把它拼进字符串。

## LIKE 的两件事不要混

- **注入**：靠 `?` 解决，输入里的引号天然安全。
- **匹配语义**：`%` / `_` 是通配符，用户输入 `a_b` 会匹配 `axb`。这是正确性问题，用
  `utils.SQLUtil.LikeContains(...)` 转义，并配 `escape '\'`。
  反过来，`escape '\'` 本身是**语法常量**（我们自己写的），不是用户输入，不需要绑定。

## 本库现状

- `PKFieldsStr()` 的返回值只用于缓存 key，不进 SQL；`where` 条件一律用绑定参数。
- `Table(ctx, name)` 的表名来自 model 的 `TableName()`，是代码常量，不接受请求参数。
  给自定义 selector 传表名时同样遵守：**表名只能来自白名单/常量**。
