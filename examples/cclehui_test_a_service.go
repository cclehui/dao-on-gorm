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
//
// 三个写方法都能接上层事务, 只是 Option 类型按阶段分开了:
// Create / Delete 是 `daoongorm.Option`(要构造新 DAO), Update 是 `daoongorm.UpdateOption`
// (DAO 已经 Load 好, 只该换 tx). 上层事务直接透传:
//
//	dbClient.Transaction(ctx, func(ctx context.Context, tx *daoongorm.DBClient) error {
//		_, err := service.Create(ctx, params, daoongorm.OptionSetTX(tx))
//		return err
//	})
//
// 这样 service 的写和上层其它语句落在同一个事务里, 一起提交或一起回滚.
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
//
// 事务内调用传 daoongorm.OptionSetTX(tx), 插入才会和上层其它语句一起提交/回滚。
func (service *CclehuiTestAService) Create(ctx context.Context, params *CreateCclehuiTestAParams,
	options ...daoongorm.Option) (*model.CclehuiTestADao, error) {
	if params == nil {
		return nil, errors.New("CreateCclehuiTestAParams must not be nil")
	}

	extraStr, err := params.Extra.ToJSON()
	if err != nil {
		return nil, err
	}

	// 这个 DAO 是 service 自己造的, 可写与否由 service 决定:
	// 调用方的 Option 在前, 这里补的在后覆盖它, 免得调用方传了只读把写接口变成"永远失败".
	options = append(options, daoongorm.OptionSetReadOnly(false))

	newDao, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{}, options...)
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
//   - 未 Load 的 DAO, isNewRow 默认就是 true, UpSert 会走 Create, "更新不存在的数据"会静默变成插入;
//   - 用 &Dao{ID: x, Name: "y"} 这种只填几列的结构体去更新, 没填的列等于零值,
//     Age 这种时间列会被 MySQL 当成 '0000-00-00' 报 1292。
//
// 这些校验都在 DaoBase.Update 里, service 只负责把 Option 透传过去, 不再重复判断一遍 ——
// "能不能更新"由 DAO 自己的状态决定, 抄一遍只会多一份迟早会走样的规则。
//
// 事务内更新时, Load 和 Update 要带同一个 tx:
//
//	row, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: id},
//		daoongorm.OptionSetReadOnly(false), daoongorm.OptionSetTX(tx))
//	// 改字段...
//	_, err = service.Update(ctx, row, daoongorm.UpdateOptionSetTX(tx))
func (service *CclehuiTestAService) Update(ctx context.Context, updateDao *model.CclehuiTestADao,
	options ...daoongorm.UpdateOption) (*model.CclehuiTestADao, error) {
	if updateDao == nil {
		return nil, errors.New("updateDao must not be nil")
	}

	daoBase := updateDao.GetDaoBase()
	if daoBase == nil {
		return nil, errors.New("updateDao has no DaoBase, load it with NewCclehuiTestADao first")
	}

	if err := daoBase.Update(ctx, options...); err != nil {
		return nil, errors.Wrapf(err, "update failed, id:%d", updateDao.ID)
	}

	return updateDao, nil
}

// Delete 按主键删除. 删除不存在的行不报错(与 DaoBase.Delete 的语义保持一致).
//
// 事务内调用传 daoongorm.OptionSetTX(tx), 删除才会和上层其它语句一起提交/回滚。
func (service *CclehuiTestAService) Delete(ctx context.Context, dataID int,
	options ...daoongorm.Option) error {
	if dataID == 0 {
		return errors.New("primary key ID is empty")
	}

	options = append(options, daoongorm.OptionSetReadOnly(false))

	deleteDao, err := model.NewCclehuiTestADao(ctx, &model.CclehuiTestADao{ID: dataID}, options...)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := deleteDao.GetDaoBase().Delete(ctx); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
