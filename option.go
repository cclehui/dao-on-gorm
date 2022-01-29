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
