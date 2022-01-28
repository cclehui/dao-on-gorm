# dao-on-gorm
基于gorm 的数据操作封装，cache 缓存、事务支持

## 特殊功能说明
1. 自动传入gin.Ctx, 根据body json格式参数是否传某个键值自动填充dao中的数据, 见 gin_util.go -> UpdateDaoByGinCtxJSON
2. 自动转map view , 其中对time.Time类型自动转换成  2006-01-02 15:04:05  格式 , daoBase.GetDefaultMapView()
3. 提供获取一个库一个表某行记录的唯一字符串ID  , daoBase.GetUniqKey()

## 修改说明
2021-07-30 增加数据库中没有记录的情况下，读取也可以强制缓存功能, OptionNewForceCache(true) 即可


## 测试表结构
```

CREATE TABLE `cclehui_test_a` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `version` int(10) unsigned NOT NULL DEFAULT '99',
  `weight` decimal(10,2) unsigned NOT NULL DEFAULT '0.00',
  `age` datetime NOT NULL DEFAULT '1970-01-01 00:00:00',
  `extra` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='测试表a';

// 非ID主键

CREATE TABLE `cclehui_test_b` (
  `column_id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `version` int(10) unsigned NOT NULL DEFAULT '99',
  `weight` decimal(10,2) unsigned NOT NULL DEFAULT '0.00',
  `age` datetime NOT NULL DEFAULT '1970-01-01 00:00:00',
  `extra` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '更新时间',
  PRIMARY KEY (`column_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='测试表a';

// 非自增主键
CREATE TABLE `cclehui_test_c` (
  `column_id` int(10) unsigned NOT NULL COMMENT 'id',
  `version` int(10) unsigned NOT NULL DEFAULT '99',
  `weight` decimal(10,2) unsigned NOT NULL DEFAULT '0.00',
  `age` datetime NOT NULL DEFAULT '1970-01-01 00:00:00',
  `extra` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '更新时间',
  PRIMARY KEY (`column_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='测试表a';

// 联合主键
CREATE TABLE `cclehui_test_d` (
  `user_id` int(10) unsigned NOT NULL COMMENT 'id',
  `column_id` int(10) unsigned NOT NULL COMMENT 'id',
  `version` int(10) unsigned NOT NULL DEFAULT '99',
  `weight` decimal(10,2) unsigned NOT NULL DEFAULT '0.00',
  `age` datetime NOT NULL DEFAULT '1970-01-01 00:00:00',
  `extra` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT '1970-01-01 00:00:00' COMMENT '更新时间',
  PRIMARY KEY (`user_id`,`column_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='测试表a';
```

## 测试 dao实现定义例子

###
1.primaryKey;primary_key  标识主键, 有两个是兼容gorm v1 和v2
2.column_default 列的默认值
3.CreatedAt ,UpdatedAt 会自动填充, 修改的之后可以指定 UpdatedAt , 这时候不会自动填充

```
type CclehuiTestADao struct {
	// ID int `gorm:"column:id;primaryKey" structs:"id" json:"id"`
	// UserID    int       `gorm:"column:user_id;primaryKey" structs:"user_id" json:"user_id"`
	ColumnID  int       `gorm:"column:column_id;primaryKey;primary_key" structs:"column_id" json:"column_id"`
	Version   int64     `gorm:"column:version" structs:"version" json:"version"`
	Weight    float64   `gorm:"column:weight;column_default:1.9" structs:"weight" json:"weight"`
	Age       time.Time `gorm:"column:age" structs:"age" json:"age"`
	Extra     string    `gorm:"column:extra" structs:"extra" json:"extra"`
	CreatedAt time.Time `gorm:"column:created_at" structs:"created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" structs:"updated_at" json:"updated_at"`

	daoBase *DaoBase
}

func init() {
	RegisterModel(&CclehuiTestADao{})
}

func NewCclehuiTestADao(ctx context.Context, myDao *CclehuiTestADao, readOnly bool) (*CclehuiTestADao, error) {
	daoBase, err := NewDaoBase(ctx, myDao, readOnly)

	myDao.daoBase = daoBase

	return myDao, err
}

// 支持事务
func NewCclehuiTestADaoWithTX(ctx context.Context,
	myDao *CclehuiTestADao, tx *sql.OrmDB) (*CclehuiTestADao, error) {

	daoBase, err := NewDaoBaseWithTX(ctx, myDao, tx)

	myDao.daoBase = daoBase

	return myDao, err
}

func (myDao *CclehuiTestADao) DBName() string {
	return "fmot"
}

func (myDao *CclehuiTestADao) TableName() string {
	// return "cclehui_test_a"
	return "cclehui_test_b"
	// return "cclehui_test_c"
	// return "cclehui_test_d"
}

func (myDao *CclehuiTestADao) RedisPool() *redis.Pool {
	return global.GetDao().GetRedisCarAPI()
}

func (myDao *CclehuiTestADao) DBClient() *sql.OrmDB {
	return global.GetDao().GetMysqlFmot()
}

func (myDao *CclehuiTestADao) Create(ctx context.Context) error {
	return myDao.daoBase.Create(ctx)
}

func (myDao *CclehuiTestADao) Update(ctx context.Context) error {
	return myDao.daoBase.Update(ctx)
}

func (myDao *CclehuiTestADao) Save(ctx context.Context) error {
	if myDao.IsNewRow() {
		return myDao.Create(ctx)
	}

	return myDao.Update(ctx)
}

func (myDao *CclehuiTestADao) Delete(ctx context.Context) error {
	return myDao.daoBase.Delete(ctx)
}

func (myDao *CclehuiTestADao) IsNewRow() bool {
	return myDao.daoBase.IsNewRow()
}

func (myDao *CclehuiTestADao) UseCache() bool {
	return true
}


```

## 测试的controler 代码参考

```
type ThemeController struct{}

func (controller *ThemeController) Test(ctx *gin.Context) {

	// 表1

	/*
					testDao := &dataversion.CclehuiTestADao{}
					testDao.Age = time.Now()
					testDao, _ = dataversion.NewCclehuiTestADao(ctx, testDao, false)
					err := testDao.Save(ctx)

					fmt.Printf("创建记录, err:%+v, %+v\n", err, testDao)


				testDao := &dataversion.CclehuiTestADao{ID: 1}
				testDao, err := dataversion.NewCclehuiTestADao(ctx, testDao, true)
				fmt.Printf("查询记录, err:%+v, %+v\n", err, testDao)

			testDao := &dataversion.CclehuiTestADao{ID: 1}
			testDao, err := dataversion.NewCclehuiTestADao(ctx, testDao, false)
			fmt.Printf("修改前, err:%+v, %+v\n", err, testDao)
			 testDao.Version = 100
			 testDao.Extra = "xxxxx"
			testDao.Version = 0
			testDao.Extra = ""
			err = testDao.Save(ctx)
			fmt.Printf("修改后, err:%+v, %+v\n", err, testDao)
		testDao := &dataversion.CclehuiTestADao{ID: 1}
		testDao, err := dataversion.NewCclehuiTestADao(ctx, testDao, false)
		fmt.Printf("删除前, err:%+v, %+v\n", err, testDao)
		err = testDao.Delete(ctx)
		fmt.Printf("删除后, err:%+v, %+v\n", err, testDao)
	*/

	// 表2 主键非ID命名
	/*
			testDao := &dataversion.CclehuiTestADao{}
			testDao.Age = time.Now()
			testDao, _ = dataversion.NewCclehuiTestADao(ctx, testDao, false)
			testDao.DisableCache()
			err := testDao.Save(ctx)

		fmt.Printf("创建记录, 表2, err:%+v, %+v\n", err, testDao)
	*/

	id := ginextend.QueryInt(ctx, "id")
	readOnly := ginextend.QueryBool(ctx, "read_only")
	deleteData := ginextend.QueryBool(ctx, "delete_data")
	addData := ginextend.QueryBool(ctx, "add_data")
	testDao := &dataversion.CclehuiTestADao{ColumnID: id}
	testDao, _ = dataversion.NewCclehuiTestADao(ctx, testDao, readOnly)
	testDao.Age = time.Now()

	if testDao.IsNewRow() {
		fmt.Printf("记录不存在\n")
		if addData {
			err := testDao.Save(ctx)

			fmt.Printf("创建记录, 表2, err:%+v, %+v\n", err, testDao)
		}
	} else {
		if readOnly {
			fmt.Printf("查询记录, %+v\n", testDao)
		} else if deleteData {
			err := testDao.Delete(ctx)

			fmt.Printf("删除记录, err:%+v, %+v\n", err, testDao)

		} else {
			err := testDao.Save(ctx)

			fmt.Printf("更新记录, err:%+v, %+v\n", err, testDao)
		}
	}

	// 非自增主键
	/*
		testDao := &dataversion.CclehuiTestADao{ColumnID: 10}
		testDao.Age = time.Now()
		testDao, _ = dataversion.NewCclehuiTestADao(ctx, testDao, false)
		err := testDao.Save(ctx)
		fmt.Printf("创建记录, 表3, 非自增主键, err:%+v, %+v\n", err, testDao)
	*/

	// 联合主键
	/*
		testDao := &dataversion.CclehuiTestADao{UserID: 1001, ColumnID: 10}
		testDao.Age = time.Now()
		// testDao, _ = dataversion.NewCclehuiTestADao(ctx, testDao, false)
		// err := testDao.Save(ctx)
		// fmt.Printf("创建记录, 表4, 联合主键, err:%+v, %+v\n", err, testDao)
		testDao, err := dataversion.NewCclehuiTestADao(ctx, testDao, true)
		fmt.Printf("查询记录, 表4, 联合主键, err:%+v, %+v\n", err, testDao)
	*/

	response.StandardJSON(ctx, "aaaa", nil)

}

```
