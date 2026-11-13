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

// UpdateOption DaoBase.Update 专用的 Option.
//
// 有意收窄到"只能绑定事务": Update 作用在已经 Load 好的 DAO 上,
// 只读、缓存、时间字段名都是 Load 阶段就决定了的, 这时候再改只会让人误以为生效.
// 接口方法不导出, 外部包实现不了, 以后也塞不进别的配置.
type UpdateOption interface {
	applyUpdateOption(daoBase *DaoBase)
}

type updateOptionFunc func(daoBase *DaoBase)

func (optionFunc updateOptionFunc) applyUpdateOption(daoBase *DaoBase) {
	optionFunc(daoBase)
}

// UpdateOptionSetTX 让这条 UPDATE 走指定事务.
//
// 只能传 Load 时用的那个 tx: Update 的基线(oldDataMap)和 FOR UPDATE 行锁都在 Load 阶段,
// 换成别的连接写等于拿另一条连接读到的旧值去覆盖, 调用方以为有锁其实没有.
func UpdateOptionSetTX(tx DBClientInterface) UpdateOption {
	return updateOptionFunc(func(daoBase *DaoBase) {
		daoBase.tx = tx
	})
}
