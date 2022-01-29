package dao

import (
	"context"
	"fmt"
	"testing"
	"time"

	daoongorm "github.com/cclehui/dao-on-gorm"
	"github.com/stretchr/testify/assert"
)

// 单元测试

// 不带缓存 curd
func TestCclehuiTestADao_NoCache(t *testing.T) {
	ctx := context.Background()
	logPrefix := "CclehuiTestADao"

	// 创建新记录
	testDao, err := NewCclehuiTestADao(ctx, &CclehuiTestADao{}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Extra = "字符串数据:11111aa"
	err = testDao.GetDaoBase().Save(ctx)
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

	// 更新数据 --------------------
	testDao2, err := NewCclehuiTestADao(ctx, &CclehuiTestADao{ID: testDao.ID}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))

	testDao2.Version = 21
	err = testDao2.GetDaoBase().Save(ctx)
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
	daoongorm.SetGlobalCacheUtil(GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "TestCclehuiTestADao_WithCache"

	// 创建新记录
	testDao, err := NewCclehuiTestADao(ctx, &CclehuiTestADao{}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Extra = "字符串数据:222222222"
	err = testDao.GetDaoBase().Save(ctx)
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
	testDao2, err := NewCclehuiTestADao(ctx, &CclehuiTestADao{ID: testDao.ID}, true) // 这里是true
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), false) // 不是从db中加载， 从缓存中加载的

	fmt.Printf("%s, 缓存检测成功\n", logPrefix)

	// 更新数据 --------------------
	testDao2, err = NewCclehuiTestADao(ctx, &CclehuiTestADao{ID: testDao.ID}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), true) // 从db中加载

	testDao2.Version = 21
	err = testDao2.GetDaoBase().Save(ctx)
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
	daoongorm.SetGlobalCacheUtil(GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "非ID为主键key的表"

	// 创建新记录
	testDao, err := NewCclehuiTestBDao(ctx, &CclehuiTestBDao{}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Extra = "字符串数据:bbbbbbbbbbbbbbb"
	err = testDao.GetDaoBase().Save(ctx)
	assert.Equal(t, err, nil)

	lastUpdatedAt := time.Now()

	fmt.Printf("%s, create 测试成功, ID:%d, : %+v\n",
		logPrefix, testDao.ColumnID, testDao.GetDaoBase().GetDefaultMapView())

	// 创建新记录 END -----------------

	time.Sleep(time.Second * 1)

	// 更新数据 --------------------
	testDao2, err := NewCclehuiTestBDao(ctx, &CclehuiTestBDao{ColumnID: testDao.ColumnID}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))

	err = testDao2.GetDaoBase().Save(ctx)
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
	daoongorm.SetGlobalCacheUtil(GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "TestCclehuiTestCDao_WithCache, 非自增主键"

	// 创建新记录
	testDao, err := NewCclehuiTestCDao(ctx, &CclehuiTestCDao{ColumnID: 100}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Extra = "字符串数据:cccccccccc"
	err = testDao.GetDaoBase().Save(ctx)
	assert.Equal(t, err, nil)

	// 创建成功， 确认数据正确性
	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), false) // 数据保存成功后 isNewRow变成false
	lastUpdatedAt := testDao.UpdatedAt

	fmt.Printf("%s, create 测试成功, ID:%d, : %+v\n",
		logPrefix, testDao.ColumnID, testDao.GetDaoBase().GetDefaultMapView())

	// 创建新记录 END -----------------

	time.Sleep(time.Second * 1)

	// 从缓存中Load 数据 --------------
	testDao2, err := NewCclehuiTestCDao(ctx, &CclehuiTestCDao{ColumnID: testDao.ColumnID}, true) // 这里是true
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), false) // 不是从db中加载， 从缓存中加载的

	// 更新数据 --------------------
	testDao2, err = NewCclehuiTestCDao(ctx, &CclehuiTestCDao{ColumnID: testDao.ColumnID}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), true) // 从db中加载

	testDao2.Version = 21
	err = testDao2.GetDaoBase().Save(ctx)
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
	daoongorm.SetGlobalCacheUtil(GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "TestCclehuiTestDDao_WithCache, 联合主键"

	// 创建新记录
	testDao, err := NewCclehuiTestDDao(ctx,
		&CclehuiTestDDao{UserID: 100801, ColumnID: 100}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), true)

	testDao.Version = 10
	testDao.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
	testDao.Extra = "字符串数据:cccccccccc"
	err = testDao.GetDaoBase().Save(ctx)
	assert.Equal(t, err, nil)

	// 创建成功， 确认数据正确性
	assert.Equal(t, testDao.GetDaoBase().IsNewRow(), false) // 数据保存成功后 isNewRow变成false
	lastUpdatedAt := testDao.UpdatedAt

	fmt.Printf("%s, create 测试成功, ID:%s, : %+v\n",
		logPrefix, testDao.GetDaoBase().GetUniqKey(), testDao.GetDaoBase().GetDefaultMapView())

	// 创建新记录 END -----------------

	time.Sleep(time.Second * 1)

	// 从缓存中Load 数据 --------------
	testDao2, err := NewCclehuiTestDDao(ctx,
		&CclehuiTestDDao{UserID: testDao.UserID, ColumnID: testDao.ColumnID}, true) // 这里是true
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), false) // 不是从db中加载， 从缓存中加载的

	// 更新数据 --------------------
	testDao2, err = NewCclehuiTestDDao(ctx,
		&CclehuiTestDDao{UserID: testDao.UserID, ColumnID: testDao.ColumnID}, false)
	assert.Equal(t, err, nil)

	assert.Equal(t, testDao2.GetDaoBase().IsNewRow(), false)
	assert.Equal(t, testDao2.Version, int64(10))
	assert.Equal(t, testDao2.GetDaoBase().IsLoadFromDB(), true) // 从db中加载

	testDao2.Version = 21
	err = testDao2.GetDaoBase().Save(ctx)
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
	daoongorm.SetGlobalCacheUtil(GetCacheUtil()) // 设置全局缓存组件
	defer func() {
		daoongorm.SetGlobalCacheUtil(&daoongorm.NopCacheUtil{})
	}()

	logPrefix := "TestCclehuiTestDao_WithTX, 事务测试"

	dbClient := GetDBClient()
	var testDaoA *CclehuiTestADao
	var testDaoB *CclehuiTestBDao
	var testDaoD *CclehuiTestDDao

	fmt.Printf("%s, 事务开始\n", logPrefix)

	var err error
	err = dbClient.Transaction(ctx, func(ctx context.Context, tx *DBClientDemo) error {
		testDaoA, _ = NewCclehuiTestADaoWithTX(ctx, &CclehuiTestADao{}, tx)
		testDaoA.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
		testDaoA.Extra = "事务数据:aaaaaaaa"
		err = testDaoA.GetDaoBase().Save(ctx)
		if err != nil {
			return err
		}

		testDaoB, _ = NewCclehuiTestBDaoWithTX(ctx, &CclehuiTestBDao{}, tx)
		testDaoB.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
		testDaoB.Extra = "事务数据:bbbbbbbbb"
		err = testDaoB.GetDaoBase().Save(ctx)
		if err != nil {
			return err
		}

		testDaoD, _ = NewCclehuiTestDDaoWithTX(ctx,
			&CclehuiTestDDao{UserID: 100801, ColumnID: 100}, tx)
		testDaoD.Age, _ = time.Parse("2006-01-02 15:04:05", "1989-03-18 10:24:32")
		testDaoD.Extra = "事务数据:ddddddddd"
		err = testDaoD.GetDaoBase().Save(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	fmt.Printf("%s, 事务结束\n", logPrefix)

	assert.Equal(t, err, nil)

	nowTime := time.Now()

	testDaoACheck, err := NewCclehuiTestADao(ctx, &CclehuiTestADao{ID: testDaoA.ID}, false)
	assert.Equal(t, err, nil)
	assert.Equal(t, testDaoACheck.GetDaoBase().IsNewRow(), false)
	assert.True(t, nowTime.Sub(testDaoACheck.UpdatedAt) < time.Second*10)
	assert.True(t, testDaoACheck.Extra != "")

	testDaoBCheck, err := NewCclehuiTestBDao(ctx, &CclehuiTestBDao{ColumnID: testDaoB.ColumnID}, false)
	assert.Equal(t, err, nil)
	assert.Equal(t, testDaoBCheck.GetDaoBase().IsNewRow(), false)
	assert.True(t, nowTime.Sub(testDaoBCheck.UpdatedAtNew) < time.Second*10)
	assert.True(t, testDaoBCheck.Extra != "")

	testDaoDCheck, err := NewCclehuiTestDDao(ctx,
		&CclehuiTestDDao{UserID: testDaoD.UserID, ColumnID: testDaoD.ColumnID}, false)
	assert.Equal(t, err, nil)
	assert.Equal(t, testDaoDCheck.GetDaoBase().IsNewRow(), false)
	assert.True(t, nowTime.Sub(testDaoDCheck.UpdatedAt) < time.Second*10)
	assert.True(t, testDaoDCheck.Extra != "")
}
