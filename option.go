package database

type Option interface {
	Apply(*DaoBase)
}

type OptionFunc func(daoBase *DaoBase)

func (of OptionFunc) Apply(daoBase *DaoBase) {
	of(daoBase)
}

// 记录不存在是否也缓存
func OptionNewForceCache(newForceCache bool) Option {
	return OptionFunc(func(daoBase *DaoBase) {
		daoBase.newForceCache = newForceCache
	})
}
