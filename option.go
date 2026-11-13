package daoongorm

type Option interface {
	Apply(*DaoBase)
}

type OptionFunc func(daoBase *DaoBase)

func (of OptionFunc) Apply(daoBase *DaoBase) {
	of(daoBase)
}

func OptionSetUseCache(useCache bool) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		daoBase.useCache = useCache
	})
}

// 设置缓存组件
func OptionSetCacheUtil(cacheUtil CacheInterface) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		daoBase.cacheUtil = cacheUtil
	})
}

func OptionSetCacheExpireTS(expireTS int) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		daoBase.cacheExpireTS = expireTS
	})
}

// 记录不存在是否也缓存
func OptionNewForceCache(newForceCache bool) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		daoBase.newForceCache = newForceCache
	})
}

func OptionSetFieldNameCreatedAt(fieldName string) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		if fieldName == "" {
			return
		}
		daoBase.fieldNameCreatedAt = fieldName
	})
}

func OptionSetFieldNameUpdatedAt(fieldName string) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		if fieldName == "" {
			return
		}
		daoBase.fieldNameUpdatedAt = fieldName
	})
}

// OptionSetReadOnly 只读模式: 写操作直接报错. NewDaoBase 的默认值是 true, 要写需显式关掉.
func OptionSetReadOnly(readOnly bool) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		daoBase.isReadOnly = readOnly
	})
}

// OptionSetTX 绑定事务连接. 传入后所有读写都走这个 tx, 不再用 model.DBClient().
//
// 事务里的读写必须用同一个连接, 否则读到的可能是别的连接上未提交/已提交的数据.
func OptionSetTX(tx DBClientInterface) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		daoBase.tx = tx
	})
}
