package dao

import (
	"context"
	"time"

	daoongorm "github.com/cclehui/dao-on-gorm"
	"github.com/cclehui/dao-on-gorm/examples/model"
	"github.com/pkg/errors"
)

// CclehuiTestAService 写操作的示例: 负责参数组装与业务校验, 真正落库仍交给 DaoBase.
//
// 和 selector 分开是因为两者职责不同: selector 只读, service 只写(且必须走主库).
// 这里不带事务; 需要多条语句原子时用 GetDBClient().Transaction, 见 dao_test.go。
type CclehuiTestAService struct{}

// CreateCclehuiTestAParams 创建参数.
//
// 主键不自增也不需要调用方传; Extra 用结构体而不是 JSON 字符串, 由 service 负责编码,
// 这样调用方不用关心 "空串不是合法 JSON" 这个存储细节。
type CreateCclehuiTestAParams struct {
	Version int64
	Weight  float64
	Age     time.Time
	Name    string
	Extra   *model.CclehuiTestAExtra
}

// Create 新建一行, 返回落库后的数据(含自增 ID 与自动填充的时间).
func (service *CclehuiTestAService) Create(ctx context.Context, params *CreateCclehuiTestAParams) (
	*model.CclehuiTestADao, error) {
	if params == nil {
		return nil, errors.New("CreateCclehuiTestAParams must not be nil")
	}

	extraStr, err := params.Extra.ToJSON()
	if err != nil {
		return nil, err
	}

	newDao, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{}, daoongorm.OptionSetReadOnly(false))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	newDao.Version = params.Version
	newDao.Weight = params.Weight
	newDao.Age = params.Age
	newDao.Name = params.Name
	newDao.Extra = extraStr

	if err := newDao.GetDaoBase().Create(ctx); err != nil {
		return nil, errors.WithStack(err)
	}

	return newDao, nil
}

// Update 按主键更新传入的 DAO.
//
// 调用方必须传入已经 Load 过的可写 DAO(NewCclehuiTestADao(ctx, &Dao{ID: x}, OptionSetReadOnly(false))),
// 因为 DaoBase.Update 的基线是 Load 那一刻的库值: 只改想改的字段即可, 没改的列不会被写,
// 零值也能表达("把 version 改成 0")。
//
// 不要用 selector 查出来的 DAO: 它是只读的(而且读的是副本, 基线可能已经过期), 写不进去。
//
// 为什么要求先 Load:
//   - 未 Load 的 DAO, Save 会认为是新行并走 Create, "更新不存在的数据"会静默变成插入;
//   - 用 &Dao{ID: x, Name: "y"} 这种只填几列的结构体去更新, 没填的列等于零值,
//     Age 这种时间列会被 MySQL 当成 '0000-00-00' 报 1292。
//
// 这里的检查只是兜底, 正确的用法是调用方从库里读出来再改。
func (service *CclehuiTestAService) Update(ctx context.Context, updateDao *model.CclehuiTestADao) (
	*model.CclehuiTestADao, error) {
	if updateDao == nil {
		return nil, errors.New("updateDao must not be nil")
	}

	if updateDao.ID == 0 {
		return nil, errors.New("primary key ID is empty")
	}

	daoBase := updateDao.GetDaoBase()
	if daoBase == nil {
		return nil, errors.New("updateDao has no DaoBase, load it with NewCclehuiTestADao first")
	}

	if daoBase.IsNewRow() {
		return nil, errors.Errorf("row not found, id:%d", updateDao.ID)
	}

	if !daoBase.IsLoadFromDB() {
		return nil, errors.New("updateDao was not loaded from db, Load it before Update")
	}

	// 只读 DAO(selector 查出来的、或者 NewCclehuiTestADao 不传 OptionSetReadOnly(false))不能写.
	// selector 走的是读副本, 用它做基线还可能读到旧值, 所以更新必须走可写的 Load.
	if err := daoBase.Save(ctx); err != nil {
		return nil, errors.Wrapf(err,
			"update failed, make sure the DAO was loaded for write (NewCclehuiTestADao(ctx, dao, OptionSetReadOnly(false))), id:%d",
			updateDao.ID)
	}

	return updateDao, nil
}

// Delete 按主键删除. 删除不存在的行不报错(与 DaoBase.Delete 的语义保持一致).
func (service *CclehuiTestAService) Delete(ctx context.Context, dataID int) error {
	if dataID == 0 {
		return errors.New("primary key ID is empty")
	}

	deleteDao, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: dataID}, daoongorm.OptionSetReadOnly(false))
	if err != nil {
		return errors.WithStack(err)
	}

	if err := deleteDao.GetDaoBase().Delete(ctx); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
