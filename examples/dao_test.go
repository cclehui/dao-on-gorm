package dao

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	daoongorm "github.com/cclehui/dao-on-gorm"
	"github.com/cclehui/dao-on-gorm/examples/global"
	"github.com/cclehui/dao-on-gorm/examples/model"
	"github.com/stretchr/testify/assert"
)

// 单元测试

// 不带缓存 curd
func TestCclehuiTestADao_NoCache(t *testing.T) {
	ctx := context.Background()
	logPrefix := "model.CclehuiTestADao"

	// 创建新记录
	testDao, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Name = "A 测试"
	testDao.Extra = `{"tag":"字符串数据:11111aa","score":1}`
	err = testDao.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	// 创建成功， 确认数据正确性
	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), false) // 数据保存成功后 isNewRow变成false
	assert.Equal(t, testDao.Weight, 1.9)                    // column_default tag 测试
	lastUpdatedAt := testDao.UpdatedAt

	// 从库中通过ID查询数据
	selector := &CclehuiTestASelector{}
	dataCount, err := selector.GetDataCountByID(ctx, testDao.ID)
	assert.Equal(t, err, nil)
	assert.Equal(t, dataCount, int64(1))

	// extra 是 JSON 列, 由 GetExtra 手工解码成结构体
	extra, err := testDao.GetExtra()
	assert.Equal(t, err, nil)
	assert.Equal(t, extra.Tag, "字符串数据:11111aa")
	assert.Equal(t, extra.Score, 1)

	fmt.Printf("%s, create 测试成功, ID:%d, : %+v\n",
		logPrefix, testDao.ID, testDao.GetDaoBase().GetDefaultMapView())

	// 创建新记录 END -----------------

	time.Sleep(time.Second * 1)

	// 更新数据 --------------------
	testDao2, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: testDao.ID}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))

	testDao2.Version = 21
	err = testDao2.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	assert.True(t, testDao2.UpdatedAt.Sub(lastUpdatedAt) >= time.Second) // udpate_at自动更新
	fmt.Printf("%s, update 测试成功\n", logPrefix)

	// 删除数据 --------------------
	err = testDao2.GetDaoBase().Delete(ctx)
	assert.Equal(t, err, nil)

	dataCount, err = selector.GetDataCountByID(ctx, testDao.ID)
	assert.Equal(t, err, nil)
	assert.Equal(t, dataCount, int64(0))

	fmt.Printf("%s, delete 测试成功\n", logPrefix)
}

// 带缓存 curd
func TestCclehuiTestADao_WithCache(t *testing.T) {
	ctx := context.Background()
	daoongorm.SetGlobalCacheUtil(global.GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "TestCclehuiTestADao_WithCache"

	// 创建新记录
	testDao, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Name = "A 缓存测试"
	testDao.Extra = `{"tag":"字符串数据:222222222","score":2}`
	err = testDao.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	// 创建成功， 确认数据正确性
	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), false) // 数据保存成功后 isNewRow变成false
	assert.Equal(t, testDao.Weight, 1.9)                    // column_default tag 测试
	lastUpdatedAt := testDao.UpdatedAt

	// 从库中通过ID查询数据
	selector := &CclehuiTestASelector{}
	dataCount, err := selector.GetDataCountByID(ctx, testDao.ID)
	assert.Equal(t, err, nil)
	assert.Equal(t, dataCount, int64(1))

	fmt.Printf("%s, create 测试成功, ID:%d, : %+v\n",
		logPrefix, testDao.ID, testDao.GetDaoBase().GetDefaultMapView())

	// 创建新记录 END -----------------

	time.Sleep(time.Second * 1)

	// 从缓存中Load 数据 --------------
	testDao2, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: testDao.ID}) // 这里是true
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), false) // 不是从db中加载， 从缓存中加载的

	fmt.Printf("%s, 缓存检测成功\n", logPrefix)

	// 更新数据 --------------------
	testDao2, err = model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: testDao.ID}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), true) // 从db中加载

	testDao2.Version = 21
	err = testDao2.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	assert.True(t, testDao2.UpdatedAt.Sub(lastUpdatedAt) >= time.Second) // udpate_at自动更新
	fmt.Printf("%s, update 测试成功\n", logPrefix)

	// 删除数据 --------------------
	err = testDao2.GetDaoBase().Delete(ctx)
	assert.Equal(t, err, nil)

	dataCount, err = selector.GetDataCountByID(ctx, testDao.ID)
	assert.Equal(t, err, nil)
	assert.Equal(t, dataCount, int64(0))
	fmt.Printf("%s, delete 测试成功\n", logPrefix)
}

// 非ID主键
func TestCclehuiTestBDao_WithCache(t *testing.T) {
	ctx := context.Background()
	daoongorm.SetGlobalCacheUtil(global.GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "非ID为主键key的表"

	// 创建新记录
	testDao, err := model.NewCclehuiTestBDao(ctx, &model.CclehuiTestBDao{}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Extra = "字符串数据:bbbbbbbbbbbbbbb"
	err = testDao.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	lastUpdatedAt := time.Now()

	fmt.Printf("%s, create 测试成功, ID:%d, : %+v\n",
		logPrefix, testDao.ColumnID, testDao.GetDaoBase().GetDefaultMapView())

	// 创建新记录 END -----------------

	time.Sleep(time.Second * 1)

	// 更新数据 --------------------
	testDao2, err := model.NewCclehuiTestBDao(ctx, &model.CclehuiTestBDao{ColumnID: testDao.ColumnID}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))

	err = testDao2.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	assert.True(t, testDao2.UpdatedAtNew.Sub(lastUpdatedAt) >= time.Second) // udpate_at自动更新
	fmt.Printf("%s, 自定义created_at,upadted_at字段名 测试成功\n", logPrefix)

	// 删除数据 --------------------
	err = testDao2.GetDaoBase().Delete(ctx)
	assert.Equal(t, err, nil)
}

// 非自增主键
func TestCclehuiTestCDao_WithCache(t *testing.T) {
	ctx := context.Background()
	daoongorm.SetGlobalCacheUtil(global.GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "TestCclehuiTestCDao_WithCache, 非自增主键"

	// 创建新记录
	testDao, err := model.NewCclehuiTestCDao(ctx, &model.CclehuiTestCDao{ColumnID: 100}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Extra = "字符串数据:cccccccccc"
	err = testDao.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	// 创建成功， 确认数据正确性
	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), false) // 数据保存成功后 isNewRow变成false
	lastUpdatedAt := testDao.UpdatedAt

	fmt.Printf("%s, create 测试成功, ID:%d, : %+v\n",
		logPrefix, testDao.ColumnID, testDao.GetDaoBase().GetDefaultMapView())

	// 创建新记录 END -----------------

	time.Sleep(time.Second * 1)

	// 从缓存中Load 数据 --------------
	testDao2, err := model.NewCclehuiTestCDao(ctx, &model.CclehuiTestCDao{ColumnID: testDao.ColumnID}) // 这里是true
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), false) // 不是从db中加载， 从缓存中加载的

	// 更新数据 --------------------
	testDao2, err = model.NewCclehuiTestCDao(ctx, &model.CclehuiTestCDao{ColumnID: testDao.ColumnID}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), true) // 从db中加载

	testDao2.Version = 21
	err = testDao2.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	assert.True(t, testDao2.UpdatedAt.Sub(lastUpdatedAt) >= time.Second) // udpate_at自动更新
	fmt.Printf("%s, update 测试成功\n", logPrefix)

	// 删除数据 --------------------
	err = testDao2.GetDaoBase().Delete(ctx)
	assert.Equal(t, err, nil)
}

// 联合主键
func TestCclehuiTestDDao_WithCache(t *testing.T) {
	ctx := context.Background()
	daoongorm.SetGlobalCacheUtil(global.GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "TestCclehuiTestDDao_WithCache, 联合主键"

	// 创建新记录
	testDao, err := model.NewCclehuiTestDDao(ctx,
		&model.CclehuiTestDDao{UserID: 100801, ColumnID: 101}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Extra = "字符串数据:cccccccccc"
	err = testDao.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	// 创建成功， 确认数据正确性
	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), false) // 数据保存成功后 isNewRow变成false
	lastUpdatedAt := testDao.UpdatedAt

	fmt.Printf("%s, create 测试成功, ID:%s, : %+v\n",
		logPrefix, testDao.GetDaoBase().GetUniqKey(), testDao.GetDaoBase().GetDefaultMapView())

	// 创建新记录 END -----------------

	time.Sleep(time.Second * 1)

	// 从缓存中Load 数据 --------------
	testDao2, err := model.NewCclehuiTestDDao(ctx,
		&model.CclehuiTestDDao{UserID: testDao.UserID, ColumnID: testDao.ColumnID}) // 这里是true
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), false) // 不是从db中加载， 从缓存中加载的

	// 更新数据 --------------------
	testDao2, err = model.NewCclehuiTestDDao(ctx,
		&model.CclehuiTestDDao{UserID: testDao.UserID, ColumnID: testDao.ColumnID}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), true) // 从db中加载

	testDao2.Version = 21
	err = testDao2.GetDaoBase().UpSert(ctx)
	assert.Equal(t, err, nil)

	assert.True(t, testDao2.UpdatedAt.Sub(lastUpdatedAt) >= time.Second) // udpate_at自动更新
	fmt.Printf("%s, update 测试成功\n", logPrefix)

	// 删除数据 --------------------
	err = testDao2.GetDaoBase().Delete(ctx)
	assert.Equal(t, err, nil)
}

// 事务功能单元测试
func TestCclehuiTestDao_WithTX(t *testing.T) {
	ctx := context.Background()
	daoongorm.SetGlobalCacheUtil(global.GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "TestCclehuiTestDao_WithTX, 事务测试"

	dbClient := global.GetDBClient()
	var testDaoA *model.CclehuiTestADao
	var testDaoB *model.CclehuiTestBDao
	var testDaoD *model.CclehuiTestDDao

	fmt.Printf("%s, 事务开始\n", logPrefix)

	var err error
	err = dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
		testDaoA, _ = model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{},
			daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
		testDaoA.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
		testDaoA.Name = "事务 A"
		testDaoA.Extra = `{"tag":"事务数据:aaaaaaaa"}`
		err = testDaoA.GetDaoBase().UpSert(ctx)
		if err != nil {
			return err
		}

		testDaoB, _ = model.NewCclehuiTestBDao(ctx, &model.CclehuiTestBDao{},
			daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
		testDaoB.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
		testDaoB.Extra = "事务数据:bbbbbbbbb"
		err = testDaoB.GetDaoBase().UpSert(ctx)
		if err != nil {
			return err
		}

		testDaoD, _ = model.NewCclehuiTestDDao(ctx,
			&model.CclehuiTestDDao{UserID: 100801, ColumnID: 100},
			daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
		testDaoD.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
		testDaoD.Extra = "事务数据:ddddddddd"
		err = testDaoD.GetDaoBase().UpSert(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	fmt.Printf("%s, 事务结束\n", logPrefix)

	assert.Equal(t, err, nil)

	nowTime := time.Now()

	testDaoACheck, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: testDaoA.ID}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)
	assert.Equal(t, testDaoACheck.GetDaoBase().IsNewRow(), false)
	assert.True(t, nowTime.Sub(testDaoACheck.UpdatedAt) < time.Second*10)
	assert.True(t, testDaoACheck.Extra != "")

	testDaoBCheck, err := model.NewCclehuiTestBDao(ctx, &model.CclehuiTestBDao{ColumnID: testDaoB.ColumnID}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)
	assert.Equal(t, testDaoBCheck.GetDaoBase().IsNewRow(), false)
	assert.True(t, nowTime.Sub(testDaoBCheck.UpdatedAtNew) < time.Second*10)
	assert.True(t, testDaoBCheck.Extra != "")

	testDaoDCheck, err := model.NewCclehuiTestDDao(ctx,
		&model.CclehuiTestDDao{UserID: testDaoD.UserID, ColumnID: testDaoD.ColumnID}, daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)
	assert.Equal(t, testDaoDCheck.GetDaoBase().IsNewRow(), false)
	assert.True(t, nowTime.Sub(testDaoDCheck.UpdatedAt) < time.Second*10)
	assert.True(t, testDaoDCheck.Extra != "")
}

// 列表查询：游标分页 + name 精确匹配 / 模糊匹配
func TestCclehuiTestASelector_GetDataList(t *testing.T) {
	ctx := context.Background()

	names := []string{"list_alpha", "list_axb", "list_beta", "other_alpha"}
	var ids []int
	for _, name := range names {
		testDao, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{}, daoongorm.OptionSetReadOnly(false))
		assert.Equal(t, err, nil)
		testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
		testDao.Name = name
		testDao.Extra = `{"tag":"list"}`
		assert.Equal(t, testDao.GetDaoBase().UpSert(ctx), nil)
		ids = append(ids, testDao.ID)
	}

	selector := &CclehuiTestASelector{}

	// 不带过滤条件：能查到刚插入的行, 且按 id 升序
	all, err := selector.GetDataList(ctx, &GetDataListParams{})
	assert.Equal(t, err, nil)
	assert.True(t, len(all) >= len(names))
	for i := 1; i < len(all); i++ {
		assert.True(t, all[i-1].ID < all[i].ID)
	}
	for _, item := range all {
		assert.NotEqual(t, item.GetDaoBase(), nil) // 每行都补了只读 daoBase
	}

	// name 精确匹配
	exact, err := selector.GetDataList(ctx, &GetDataListParams{Name: "list_beta"})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(exact), 1)
	assert.Equal(t, exact[0].Name, "list_beta")

	// name 模糊匹配
	like, err := selector.GetDataList(ctx, &GetDataListParams{NameLike: "alpha"})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(like), 2)

	// 下划线是 LIKE 通配符, 转义后 "axb" 不能被 "a_b" 匹配到
	raw, err := selector.GetDataList(ctx, &GetDataListParams{NameLike: "list_axb"})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(raw), 1)
	assert.Equal(t, raw[0].Name, "list_axb")

	// "_" 被转义成字面量, 所以 "list_a_b" 不会再匹配 "list_axb"
	literalUnderscore, err := selector.GetDataList(ctx, &GetDataListParams{NameLike: "list_a_b"})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(literalUnderscore), 0)

	// "%" 同样按字面量处理, 不会退化成"匹配所有"
	literalPercent, err := selector.GetDataList(ctx, &GetDataListParams{NameLike: "%"})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(literalPercent), 0)

	// 游标分页：从第一行之后继续取
	page1, err := selector.GetDataList(ctx, &GetDataListParams{NameLike: "list_", Limit: 2})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(page1), 2)

	page2, err := selector.GetDataList(ctx,
		&GetDataListParams{NameLike: "list_", LastID: page1[len(page1)-1].ID, Limit: 2})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(page2), 1)
	assert.True(t, page2[0].ID > page1[len(page1)-1].ID)

	for _, id := range ids {
		testDao, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: id}, daoongorm.OptionSetReadOnly(false))
		assert.Equal(t, err, nil)
		assert.Equal(t, testDao.GetDaoBase().Delete(ctx), nil)
	}
}

// service 层写操作示例：Create / Update / Delete
func TestCclehuiTestAService_CreateUpdateDelete(t *testing.T) {
	ctx := context.Background()
	service := &CclehuiTestAService{}
	selector := &CclehuiTestASelector{}

	// Create: extra 传结构体, service 负责编码, ID/时间由库填充
	created, err := service.Create(ctx, &CreateCclehuiTestAParams{
		Version: 10,
		Weight:  2.5,
		Age:     time.Date(1990, 5, 6, 7, 8, 9, 0, time.UTC),
		Name:    "service_create",
		Extra:   &model.CclehuiTestAExtra{Tag: "svc", Score: 9},
	})
	assert.Equal(t, err, nil)
	assert.True(t, created.ID > 0)
	assert.Equal(t, created.GetDaoBase().IsNewRow(), false)
	assert.True(t, !created.CreatedAt.IsZero())

	// 先 Load(可写) 再改: 只改 name 和 extra, 其他列保持原值
	toUpdate, err := loadWritableDao(ctx, created.ID)
	assert.Equal(t, err, nil)
	toUpdate.Name = "service_updated"
	toUpdate.Version = 10
	toUpdate.Extra, err = (&model.CclehuiTestAExtra{Tag: "svc2", Score: 42}).ToJSON()
	assert.Equal(t, err, nil)

	updated, err := service.Update(ctx, toUpdate)
	assert.Equal(t, err, nil)
	assert.Equal(t, updated.Name, "service_updated")

	fromDB, err := loadReadOnlyDao(ctx, created.ID)
	assert.Equal(t, err, nil)
	assert.Equal(t, fromDB.Name, "service_updated")
	assert.Equal(t, fromDB.Version, int64(10))
	assert.Equal(t, fromDB.Weight, 2.5) // 没传 weight, 不能被零值覆盖

	extra, err := fromDB.GetExtra()
	assert.Equal(t, err, nil)
	assert.Equal(t, extra.Tag, "svc2")
	assert.Equal(t, extra.Score, 42)

	// 更新不存在的数据必须报错, 不能被 UpSert 静默降级成 INSERT
	missing, err := loadWritableDao(ctx, created.ID+100000)
	assert.Equal(t, err, nil)
	assert.Equal(t, missing.GetDaoBase().IsNewRow(), true)
	_, err = service.Update(ctx, missing)
	assert.NotEqual(t, err, nil)

	// 没 Load 过的空 DAO 也要拦下来
	_, err = service.Update(ctx, &model.CclehuiTestADao{ID: created.ID})
	assert.NotEqual(t, err, nil)

	// Delete
	assert.Equal(t, service.Delete(ctx, created.ID), nil)
	count, err := selector.GetDataCountByID(ctx, created.ID)
	assert.Equal(t, err, nil)
	assert.Equal(t, count, int64(0))

	// 重复删除不报错
	assert.Equal(t, service.Delete(ctx, created.ID), nil)

	// 参数校验
	_, err = service.Create(ctx, nil)
	assert.NotEqual(t, err, nil)
	_, err = service.Update(ctx, nil)
	assert.NotEqual(t, err, nil)
	assert.NotEqual(t, service.Delete(ctx, 0), nil)
}

// Update 接收已 Load 的 DAO: 只改想改的字段, 未改的列保持库里原值, 零值也能表达
func TestCclehuiTestAService_UpdateWithDao(t *testing.T) {
	ctx := context.Background()
	service := &CclehuiTestAService{}

	created, err := service.Create(ctx, &CreateCclehuiTestAParams{
		Version: 7,
		Weight:  3.5,
		Age:     time.Date(1991, 2, 3, 4, 5, 6, 0, time.UTC),
		Name:    "dao_param_orig",
		Extra:   &model.CclehuiTestAExtra{Tag: "orig", Score: 1},
	})
	assert.Equal(t, err, nil)
	defer service.Delete(ctx, created.ID)

	// 先 Load(可写), 只改 name
	toUpdate, err := loadWritableDao(ctx, created.ID)
	assert.Equal(t, err, nil)
	toUpdate.Name = "dao_param_new"

	updated, err := service.Update(ctx, toUpdate)
	assert.Equal(t, err, nil)
	assert.Equal(t, updated.GetDaoBase().IsNewRow(), false)

	fromDB, err := loadReadOnlyDao(ctx, created.ID)
	assert.Equal(t, err, nil)
	assert.Equal(t, fromDB.Name, "dao_param_new")
	assert.Equal(t, fromDB.Version, int64(7))
	assert.Equal(t, fromDB.Weight, 3.5)
	assert.True(t, fromDB.Age.Equal(created.Age))

	extra, err := fromDB.GetExtra()
	assert.Equal(t, err, nil)
	assert.Equal(t, extra.Tag, "orig")
	assert.Equal(t, extra.Score, 1)

	// 零值可以显式写入(这是"先 Load 再改"的收益)
	zero, err := loadWritableDao(ctx, created.ID)
	assert.Equal(t, err, nil)
	zero.Version = 0
	_, err = service.Update(ctx, zero)
	assert.Equal(t, err, nil)

	fromDB, err = loadReadOnlyDao(ctx, created.ID)
	assert.Equal(t, err, nil)
	assert.Equal(t, fromDB.Version, int64(0))

	// 没改动时 UpSert 只 touch updated_at, 不应报错
	noop, err := loadWritableDao(ctx, created.ID)
	assert.Equal(t, err, nil)
	_, err = service.Update(ctx, noop)
	assert.Equal(t, err, nil)

	// 没 Load 过的 DAO 必须报错, 不能被 UpSert 降级成 INSERT
	_, err = service.Update(ctx, &model.CclehuiTestADao{ID: created.ID})
	assert.NotEqual(t, err, nil)

	// 只读 DAO 也要拒绝(写不进去)
	readOnly, err := loadReadOnlyDao(ctx, created.ID)
	assert.Equal(t, err, nil)
	_, err = service.Update(ctx, readOnly)
	assert.NotEqual(t, err, nil)

	// 已 Load 但行不存在: IsNewRow 为 true, 同样报错
	missing, err := loadWritableDao(ctx, created.ID+100000)
	assert.Equal(t, err, nil)
	assert.Equal(t, missing.GetDaoBase().IsNewRow(), true)
	_, err = service.Update(ctx, missing)
	assert.NotEqual(t, err, nil)
}

// loadReadOnlyDao 用默认(只读)模式 Load 一行: 走缓存 + 读副本.
func loadReadOnlyDao(ctx context.Context, id int) (*model.CclehuiTestADao, error) {
	return model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: id})
}

// loadWritableDao 用可写模式 Load 一行, 供 service.Update 使用.
func loadWritableDao(ctx context.Context, id int) (*model.CclehuiTestADao, error) {
	return model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: id}, daoongorm.OptionSetReadOnly(false))
}

// 列表查询的排序与计数
func TestCclehuiTestASelector_OrderAndCount(t *testing.T) {
	ctx := context.Background()
	selector := &CclehuiTestASelector{}

	var ids []int
	for i := 0; i < 3; i++ {
		testDao, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{},
			daoongorm.OptionSetReadOnly(false))
		assert.Equal(t, err, nil)
		testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
		testDao.Name = fmt.Sprintf("order_%d", i)
		testDao.Version = int64(100 - i) // 与 id 顺序相反, 便于验证排序生效
		testDao.Extra = `{"tag":"order"}`
		assert.Equal(t, testDao.GetDaoBase().UpSert(ctx), nil)
		ids = append(ids, testDao.ID)
	}
	defer func() {
		for _, id := range ids {
			d, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: id},
				daoongorm.OptionSetReadOnly(false))
			assert.Equal(t, err, nil)
			assert.Equal(t, d.GetDaoBase().Delete(ctx), nil)
		}
	}()

	// 默认 id 升序
	asc, err := selector.GetDataList(ctx, &GetDataListParams{NameLike: "order_"})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(asc), 3)
	assert.True(t, asc[0].ID < asc[1].ID)

	// 显式降序
	desc, err := selector.GetDataList(ctx, &GetDataListParams{NameLike: "order_", OrderBy: "id", OrderDesc: true})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(desc), 3)
	assert.True(t, desc[0].ID > desc[1].ID)

	// 按其它白名单列排序
	byVersion, err := selector.GetDataList(ctx, &GetDataListParams{NameLike: "order_", OrderBy: "version"})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(byVersion), 3)
	assert.True(t, byVersion[0].Version < byVersion[1].Version)

	// 大小写与空格不敏感
	mixed, err := selector.GetDataList(ctx, &GetDataListParams{NameLike: "order_", OrderBy: " NAME "})
	assert.Equal(t, err, nil)
	assert.Equal(t, len(mixed), 3)

	// 不在白名单里的列必须报错, 不能拼进 SQL
	_, err = selector.GetDataList(ctx, &GetDataListParams{NameLike: "order_", OrderBy: "id; drop table cclehui_test_a"})
	assert.NotEqual(t, err, nil)
	_, err = selector.GetDataList(ctx, &GetDataListParams{OrderBy: "not_a_column"})
	assert.NotEqual(t, err, nil)

	// count 与 list 口径一致(同一个过滤 + 网络游标)
	count, err := selector.GetDataListCount(ctx, &GetDataListParams{NameLike: "order_"})
	assert.Equal(t, err, nil)
	assert.Equal(t, count, int64(3))

	count, err = selector.GetDataListCount(ctx, &GetDataListParams{NameLike: "order_", LastID: asc[1].ID})
	assert.Equal(t, err, nil)
	assert.Equal(t, count, int64(1))

	// 精确匹配 + 游标都要生效
	count, err = selector.GetDataListCount(ctx, &GetDataListParams{Name: "order_0"})
	assert.Equal(t, err, nil)
	assert.Equal(t, count, int64(1))

	// Limit / OrderBy 不影响 count
	count, err = selector.GetDataListCount(ctx, &GetDataListParams{NameLike: "order_", Limit: 1, OrderBy: "version", OrderDesc: true})
	assert.Equal(t, err, nil)
	assert.Equal(t, count, int64(3))

	// nil params 也可以
	count, err = selector.GetDataListCount(ctx, nil)
	assert.Equal(t, err, nil)
	assert.True(t, count >= 3)
}

// service 的三个写方法都要能接上层事务: OptionSetTX(tx) 透传后, 要么一起提交, 要么一起回滚.
func TestCclehuiTestAService_WithTX(t *testing.T) {
	ctx := context.Background()
	service := &CclehuiTestAService{}
	dbClient := global.GetDBClient()

	// Create 在事务里回滚: 事务内用同一个 tx 读得到, 提交前外面读不到
	var rolledBackID int
	err := dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
		created, err := service.Create(ctx, &CreateCclehuiTestAParams{
			Version: 1,
			Weight:  1.5,
			Age:     time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Name:    "service_tx_rollback",
			Extra:   &model.CclehuiTestAExtra{Tag: "tx"},
		}, daoongorm.OptionSetTX(tx))
		assert.Equal(t, err, nil)
		rolledBackID = created.ID

		inTx, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: rolledBackID},
			daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
		assert.Equal(t, err, nil)
		assert.Equal(t, inTx.GetDaoBase().IsNewRow(), false)

		return errors.New("rollback on purpose")
	})
	assert.NotEqual(t, err, nil)
	assert.Equal(t, probeDataCount(t, rolledBackID), int64(0))

	// Create 正常提交
	var committedID int
	err = dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
		created, err := service.Create(ctx, &CreateCclehuiTestAParams{
			Version: 2,
			Weight:  2.5,
			Age:     time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
			Name:    "service_tx_commit",
			Extra:   &model.CclehuiTestAExtra{Tag: "tx"},
		}, daoongorm.OptionSetTX(tx))

		committedID = created.ID

		return err
	})
	assert.Equal(t, err, nil)
	assert.Equal(t, probeDataCount(t, committedID), int64(1))
	defer service.Delete(ctx, committedID)

	// Update: Load 和 UpSert 用同一个 tx, 回滚后值保持不变
	toUpdate, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: committedID},
		daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)
	toUpdate.Name = "service_tx_before"
	_, err = service.Update(ctx, toUpdate)
	assert.Equal(t, err, nil)

	err = dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
		row, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: committedID},
			daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
		if err != nil {
			return err
		}

		row.Name = "service_tx_rolled_back"

		if _, err := service.Update(ctx, row, daoongorm.UpdateOptionSetTX(tx)); err != nil {
			return err
		}

		return errors.New("rollback on purpose")
	})
	assert.NotEqual(t, err, nil)

	fromDB, err := loadReadOnlyDao(ctx, committedID)
	assert.Equal(t, err, nil)
	assert.Equal(t, fromDB.Name, "service_tx_before")

	// UpSert 带 tx 走 Update 分支: 回滚后值不变
	err = dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
		row, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: committedID},
			daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
		if err != nil {
			return err
		}

		row.Name = "service_upsert_rolled_back"

		if err := row.GetDaoBase().UpSert(ctx, daoongorm.UpdateOptionSetTX(tx)); err != nil {
			return err
		}

		return errors.New("rollback on purpose")
	})
	assert.NotEqual(t, err, nil)

	fromDB, err = loadReadOnlyDao(ctx, committedID)
	assert.Equal(t, err, nil)
	assert.Equal(t, fromDB.Name, "service_tx_before")

	// UpSert 事务外提交: 值真的落库
	row, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: committedID},
		daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)
	row.Name = "service_upsert_committed"
	assert.Equal(t, row.GetDaoBase().UpSert(ctx), nil)

	fromDB, err = loadReadOnlyDao(ctx, committedID)
	assert.Equal(t, err, nil)
	assert.Equal(t, fromDB.Name, "service_upsert_committed")

	// UpSert 主键为空走 Create: options 在 Create 分支不生效, 也不会因此报错
	newRow, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{},
		daoongorm.OptionSetReadOnly(false))
	assert.Equal(t, err, nil)
	newRow.Name = "service_upsert_create"
	newRow.Age = time.Date(2002, 2, 2, 0, 0, 0, 0, time.UTC)
	newRow.Extra = `{"tag":"upsert"}`
	assert.Equal(t, newRow.GetDaoBase().UpSert(ctx), nil)
	assert.True(t, newRow.ID > 0)
	defer service.Delete(ctx, newRow.ID)

	// Load 在事务外、只给 Update 传 tx: 基线不是同一个连接, 必须报错而不是悄悄写进去
	err = dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
		row, err := loadWritableDao(ctx, committedID)
		if err != nil {
			return err
		}

		row.Name = "service_tx_wrong_load"

		_, err = service.Update(ctx, row, daoongorm.UpdateOptionSetTX(tx))

		return err
	})
	assert.NotEqual(t, err, nil)

	// 被拒的 Update 不能写库: 值还是上面 UpSert 提交的那个
	fromDB, err = loadReadOnlyDao(ctx, committedID)
	assert.Equal(t, err, nil)
	assert.Equal(t, fromDB.Name, "service_upsert_committed")

	// Delete 在事务里回滚: 行还在
	err = dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
		if err := service.Delete(ctx, committedID, daoongorm.OptionSetTX(tx)); err != nil {
			return err
		}

		return errors.New("rollback on purpose")
	})
	assert.NotEqual(t, err, nil)
	assert.Equal(t, probeDataCount(t, committedID), int64(1))

	// Delete 提交后行才真正消失
	err = dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
		return service.Delete(ctx, committedID, daoongorm.OptionSetTX(tx))
	})
	assert.Equal(t, err, nil)
	assert.Equal(t, probeDataCount(t, committedID), int64(0))
}

// probeDataCount 按主键数行, 用只读 selector 只是为了避开事务外可见性的干扰.
func probeDataCount(t *testing.T, id int) int64 {
	t.Helper()

	count, err := (&CclehuiTestASelector{}).GetDataCountByID(context.Background(), id)
	assert.Equal(t, err, nil)

	return count
}
