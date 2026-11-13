# dao-on-gorm Coding Style

约束，不是科普。行为事实在 `harness/knowledge/`。

## 文件职责

| 文件 | 放什么 |
|---|---|
| `model.go` | `Model` 接口、`ModelDef`/`Field`/`FieldTag` 解析与注册期校验 |
| `dao_base.go` | `DaoBase` 的 CRUD、缓存、时间填充、主键处理 |
| `dao_base_ext.go` | `DaoBase` 的辅助视图与 `NewDaoBaseNoLoad` |
| `dao_util.go` | 无状态的包级工具（时间比较/舍入、主键拼串、`FOR UPDATE` clause） |
| `db_client.go` | 连接、读写分离、事务、`DBClientInterface` 实现 |
| `db_client_config.go` | 连接/超时/日志级别配置结构 |
| `cache.go` / `constant.go` / `option.go` / `log.go` | 缓存接口、全局变量、`Option`、日志接口 |
| `internal/` | 不对外暴露的工具（`structs`/`mapstructure` 包装、`ctime.Duration`） |
| `examples/` | 可运行的集成示例 + 集成测试；**不是**生产代码 |
| `examples/model/` | 示例 DAO 结构体与注册；依赖 `examples/global` 拿 DBClient |
| `examples/global/` | 示例的配置、DBClient、缓存组件初始化；不 import `model`，避免成环 |
| `examples/utils/` | 示例用的无状态工具；按 `type XxxUtilType struct{}` + `var XxxUtil XxxUtilType` 定义方法 |

新增能力优先放进已有的 owner 文件；确实需要新文件时按上表的粒度拆，不要建 `util.go`。

## 命名与注释

- 导出 API 用完整名字，不要缩写：`OptionSetFieldNameCreatedAt` 而不是 `OptCreatedAt`。
- 私有 helper 是方法（`func (daoBase *DaoBase) xxx()`）而不是包级函数，除非它真的不属于任何类型
  （`roundToFieldPrecision`、`valuesEqual` 这类纯函数可以留在 `dao_util.go`）。
- 注释用中文，只解释 **为什么**，不复述代码。凡是“反直觉但必须如此”的地方，注释里要写清
  不这么做会坏什么（例如 `clause.Locking`、`Round` 而不是 `Truncate`、事务内延迟失效缓存）。
- 不要为了让注释“好看”而删掉解释原因的段落；这些段落是防止后人“简化”的护栏。

## 错误

- 库内部用 `github.com/pkg/errors`（`New` / `Wrapf` / `WithStack`），保留堆栈。
- 面向调用方的**用法错误**用 `errors.New("...")` 直接返回，例如 `readOnly`、
  `primary key ID is empty`。这是既有约定，examples 也依赖这些错误出现；不要改成 panic，
  也不要静默忽略。
- 错误信息**统一用英文**：错误会被日志、监控、上层服务直接消费。注释仍然用中文。
- 不要吞掉 DB 错误：`Create/Update/Delete/Load` 一律把 `*gorm.DB` 的 `Error` 返回给调用方。
- **缓存错误例外**：`setCache` / `deleteCache` 只记日志（`logger.Errorf`），不影响主流程返回。
  缓存不可用不能让写操作失败，这是有意的取舍。

## 日志

- 只用 `log.go` 里的 `Logger` 接口（`logger.Errorf` / `logger.Infof`），默认实现输出单行 JSON。
- 不要直接 `log.Println` / `fmt.Println`；测试和 examples 里打印属于演示输出，可以例外。
- 日志里带 key/表名等上下文，但不要带密钥、密码、完整 DSN。

## Option

- 新开关一律做成 `Option`（`OptionFunc`），在 `NewDaoBase*` 时应用，不要在方法里读全局变量。
- 默认值集中在 `DaoBase.applyOptionDefaults()`，`NewDaoBase` 和 `NewDaoBaseNoLoad` 都调用它，
  保证两条构造路径的默认行为一致。
- 只读与事务也是 Option（`OptionSetReadOnly` / `OptionSetTX`）。构造入口只有 `NewDaoBase`
  （外加不 Load 的 `NewDaoBaseNoLoad`），不要再加并列的构造函数。
- `NewDaoBase` **默认只读**，写操作必须显式 `OptionSetReadOnly(false)`；`NewXxxDao` 这类封装
  也只有一个入口，不要额外加 `NewXxxDaoWithTX` 之类的并列函数。
- 全局可变量（`DaoCacheExpire`、`DaoCachePrefix`、`DaoCachePackagePrefix`、`FieldNameCreatedAt` 等）
  只在进程启动阶段设置；不要在处理请求的过程中改它们。

## 反射

- 只在 `model.go`（注册期）和 `DaoBase` 的字段读写里用反射。
- 反射写字段前必须检查 `sf.IsValid()`、类型和 `CanSet()`；注册期已经校验过的类型也不要省。
- 禁止用 `vf.String()` 猜零值——非 string 的 Kind 会 panic（历史上踩过）。

## 测试

- 纯函数、ModelDef 解析、diff/时间逻辑：表驱动单元测试，不连库。
- 任何涉及 gorm / MySQL 行为的结论，**必须用一次性 MySQL 实测**，不要靠读文档推断
  （例如 `precision` 是否截断、MySQL 是 Round 还是 Truncate）。
- 临时验证脚本用完即删，不要留在仓库里。
