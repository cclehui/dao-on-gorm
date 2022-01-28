package daoongorm

import "context"

type CacheInterface interface {
	SetCache(ctx context.Context, key string, value interface{}, ttl int) (err error)
	GetCache(ctx context.Context, key string, value interface{}) (hit bool, err error)
}

var nopCacheUtil = &NopCacheUtil{}

type NopCacheUtil struct{}

func (ncu *NopCacheUtil) SetCache(ctx context.Context,
	key string, value interface{}, ttl int) (err error) {
	return nil
}

func (ncu *NopCacheUtil) GetCache(ctx context.Context, key string, value interface{}) (hit bool, err error) {
	return false, nil
}
