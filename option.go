package daoongorm

type Option interface {
	Apply(*DaoBase)
}

type OptionFunc func(daoBase *DaoBase)

func (of OptionFunc) Apply(daoBase *DaoBase) {
	of(daoBase)
}

// 设置缓存组件
func OptionSetCacheUtil(cacheUtil CacheInterface) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		daoBase.cacheUtil = cacheUtil
	})
}

// 记录不存在是否也缓存
func OptionNewForceCache(newForceCache bool) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		daoBase.newForceCache = newForceCache
	})
}
