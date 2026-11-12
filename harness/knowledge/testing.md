# Testing

## 两层测试

| 层 | 位置 | 依赖 | 跑法 |
|---|---|---|---|
| 纯单元 | 根目录 `*_test.go`（`model_test.go` / `dao_base_test.go` / `db_client_test.go` / `cache_test.go`） | 无 | `go test .` |
| 集成 | `examples/dao_test.go` | 真实 MySQL + redis | `go test ./examples/` |

`go test ./...` 会同时跑两层，没起容器时 examples 必然失败。只改纯逻辑时用 `go test .` 快速迭代。

## 单元测试覆盖什么

- `ModelDef` / `FieldTag` 解析，以及注册期的 panic 场景（无主键、重复列、不支持类型、非指针）。
- `changedColumnValues` / `updateAutoFillValue` / `roundTimeFields` 这类纯内存逻辑。
- 缓存 key 结构、包级前缀、`GetUniqKey` 不受影响。
- DSN 超时拼装、日志级别映射、`AddAfterCommit` 只执行一次且有序。

`newTestDao` / `newTestDaoForPrecision` 通过 `NewDaoBaseNoLoad` 构造不连库的 `DaoBase`，
用来验证内存逻辑。**不要**为了单测引入 sqlite：这里的行为（`ON DUPLICATE`、`JSON`、
`datetime(n)`、`FOR UPDATE`）都依赖 MySQL。

## 集成测试怎么跑

本机已缓存 `mysql:8.0.44` 和 `redis:5.0.14`，优先用本地镜像：

```bash
docker run -d --rm --name daoongorm-it \
  -e MYSQL_ROOT_PASSWORD=123456 -e MYSQL_DATABASE=test \
  -e MYSQL_USER=cclehui_test -e MYSQL_PASSWORD=123456 -p 13306:3306 mysql:8.0.44

docker run -d --rm --name daoongorm-redis \
  -p 6379:6379 redis:5.0.14 redis-server --requirepass 123456
```

导入表结构（`examples/sql.md` 是 markdown，去掉代码围栏再喂给 mysql）：

```bash
awk '!/^```/' examples/sql.md | docker exec -i daoongorm-it mysql -uroot -p123456 test
```

`examples/config_demo.yaml` 默认连 `127.0.0.1:3306` 且无密码；容器映射到 13306 时临时改配置，
**跑完 `git checkout -- examples/config_demo.yaml`**，不要把本地测试配置提交进去。

```bash
go test ./examples/ -v
docker stop daoongorm-it daoongorm-redis
```

## 结论必须实测

涉及 gorm / MySQL 行为的判断（`precision` 是否截断、MySQL 是 Round 还是 Truncate、
`FOR UPDATE` 是否生效、缓存是否在提交后才失效），一律写一次性 probe 测试对着真库验证，
不要凭文档或记忆下结论。验证完删掉 probe 文件。

已经踩过的坑（值得记住）：

- gorm 的 `precision:` 只影响 DDL，不会修改传入的值；
- MySQL 写 `datetime(3)` 是**四舍五入**，`.123600` 存成 `.124000`；
- `Set("gorm:query_option", "FOR UPDATE")` 在 gorm v2 无效，要用 `clause.Locking`。

## 提交前

```bash
gofmt -l .            # 必须为空
go vet ./...
go test .             # 或 go test ./...（起容器时）
```
