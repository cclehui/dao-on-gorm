## dao-on-gorm
轻量级的基于gorm 的针对单条数据操作curd封装，操作方便， 同时基于主键对数据做了自动缓存cache，大大降低数据库压力， 事务和非事务操作模式基本一致


## 使用体验
1. 在 [examples/config_demo.yaml](examples/config_demo.yaml) 下配置自己的redis 和 mysql连接信息
2. 在你的mysql库中导入sql: [examples/sql.md](examples/sql.md)
3. go test -v ./...  即可运行项目的单测，就可以看到数据操作的日志信息


```
[5.619ms] [rows:1] INSERT INTO `cclehui_test_b` (`version`,`weight`,`age`,`extra`,`created_at_new`,`updated_at_new`) VALUES (10,1.900000,'1989-03-18 10:24:32','字符串数据:bbbbbbbbbbbbbbb','2022-01-29 18:02:10.639','2022-01-29 18:02:10.639')
非ID为主键key的表, create 测试成功, ID:15, : map[age:1989-03-18 10:24:32 column_id:15 created_at_new:2022-01-29 18:02:10 extra:字符串数据:bbbbbbbbbbbbbbb updated_at_new:2022-01-29 18:02:10 version:10 weight:1.9]

[0.492ms] [rows:1] SELECT * FROM `cclehui_test_b` WHERE `column_id` = 15 LIMIT 1

[3.759ms] [rows:1] UPDATE `cclehui_test_b` SET `age`='1989-03-18 18:24:32',`created_at_new`='2022-01-29 18:02:11',`extra`='字符串数据:bbbbbbbbbbbbbbb',`updated_at_new`='2022-01-29 18:02:11.647',`version`=10,`weight`=1.900000 WHERE `column_id` = 15
非ID为主键key的表, 自定义created_at,upadted_at字段名 测试成功

[3.133ms] [rows:1] DELETE FROM `cclehui_test_b` WHERE `cclehui_test_b`.`column_id` = 15

```


examples 目录下是完成的集成例子， 其中 [examples/dao_test.go](examples/dao_test.go) 包含了curd 和事务操作的详细例子

## 集成方法
1. dao-on-gorm 本身是基于gorm的，实现上基于[CacheInterface](cache.go) 和[DBClientInterface](db_client_interface.go) 两个interface{}, 
所以接入的时候咱们需要实现这两个api才能提供真正的cache 和db操作功能
2. DBClientInterface 也提供了一种默认的实现 [DBClient](db_client.go) , 也可以直接使用
3. CacheInterface 需要咱们实现， 实现方式可以参考 [examples/cache_util_demo.go](examples/cache_util_demo.go)

具体详细建议读一下 [examples/dao_test.go](examples/dao_test.go) 即可明白

## 基本使用
```
// 创建新记录
testDao, err := NewCclehuiTestADao(ctx, &CclehuiTestADao{}, false)
testDao.Version = 10
testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
testDao.Extra = "字符串数据:11111aa"
err = testDao.GetDaoBase().Save(ctx)

// 从从库查询数据(一行搞定)
testDao, err := NewCclehuiTestADao(ctx, &CclehuiTestADao{ID:1}, true)

// 更新数据
testDao, err := NewCclehuiTestADao(ctx, &CclehuiTestADao{ID:1}, false)
testDao.Version = 21
err = testDao.GetDaoBase().Save(ctx)

// 删除数据
err = testDao.GetDaoBase().Delete(ctx)

```

## 特性说明
[option.go](option.go) 中包含了NewDao中可设置的选项

1. 是否开启缓存
2. 是否在记录不存在时也强制缓存
3. 设置created_at,updated_at 字段名 (默认是created_at，updated_at 可以修改),该字段的值会自动填充

设置全局默认的缓存组件[SetGlobalCacheUtil](cache.go)


