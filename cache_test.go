package daoongorm

import (
	"context"
	"strings"
	"testing"
)

// 默认配置下 key 必须和老版本完全一致, 否则升级会让线上缓存整体失效.
func TestCacheKeyUnchangedByDefault(t *testing.T) {
	oldPackage, oldPrefix := DaoCachePackagePrefix, DaoCachePrefix
	defer func() {
		DaoCachePackagePrefix, DaoCachePrefix = oldPackage, oldPrefix
	}()

	DaoCachePackagePrefix = ""
	DaoCachePrefix = "daocache"

	daoBase := newTestDao(t, &diffModel{ID: 7})

	if got, want := daoBase.CacheKey(), "daocache:test_db:diff_model:7"; got != want {
		t.Fatalf("CacheKey() = %q, want %q", got, want)
	}
}

func TestCacheKeyWithPackagePrefix(t *testing.T) {
	oldPackage := DaoCachePackagePrefix
	defer func() { DaoCachePackagePrefix = oldPackage }()

	SetDaoCachePackagePrefix("identity_hub")

	daoBase := newTestDao(t, &diffModel{ID: 7})

	if got, want := daoBase.CacheKey(), "identity_hub:daocache:test_db:diff_model:7"; got != want {
		t.Fatalf("CacheKey() = %q, want %q", got, want)
	}

	// GetUniqKey 是业务唯一标识, 不该被缓存前缀影响.
	if got, want := daoBase.GetUniqKey(), "test_db:diff_model:7"; got != want {
		t.Fatalf("GetUniqKey() = %q, want %q", got, want)
	}
}

func TestSetDaoCachePackagePrefixTrims(t *testing.T) {
	oldPackage := DaoCachePackagePrefix
	defer func() { DaoCachePackagePrefix = oldPackage }()

	SetDaoCachePackagePrefix("  billing  ")

	if DaoCachePackagePrefix != "billing" {
		t.Fatalf("package prefix = %q, want %q", DaoCachePackagePrefix, "billing")
	}
}

func TestSetDaoCachePackagePrefixEmptyKeepsOldShape(t *testing.T) {
	oldPackage := DaoCachePackagePrefix
	defer func() { DaoCachePackagePrefix = oldPackage }()

	SetDaoCachePackagePrefix("something")
	SetDaoCachePackagePrefix("   ")

	daoBase := newTestDao(t, &diffModel{ID: 1})

	if got := daoBase.CacheKey(); strings.HasPrefix(got, ":") || strings.Contains(got, "::") {
		t.Fatalf("empty prefix should leave no empty segment, got %q", got)
	}
}

// 前缀必须真的作用到读写缓存上, 而不只是 CacheKey 的展示.
func TestPackagePrefixUsedByCacheOperations(t *testing.T) {
	oldPackage := DaoCachePackagePrefix
	defer func() { DaoCachePackagePrefix = oldPackage }()

	recorder := &recordingCache{}
	SetGlobalCacheUtil(recorder)
	defer SetGlobalCacheUtil(&NopCacheUtil{})

	SetDaoCachePackagePrefix("svc")

	daoBase := newTestDao(t, &diffModel{ID: 42})
	daoBase.DeleteCache(context.Background())

	if len(recorder.deleted) != 1 || recorder.deleted[0] != "svc:daocache:test_db:diff_model:42" {
		t.Fatalf("deleted keys = %v", recorder.deleted)
	}
}

type recordingCache struct {
	deleted []string
}

func (c *recordingCache) Set(context.Context, string, interface{}, int) error { return nil }
func (c *recordingCache) Get(context.Context, string, interface{}) (bool, error) {
	return false, nil
}

func (c *recordingCache) Del(_ context.Context, key string) error {
	c.deleted = append(c.deleted, key)

	return nil
}
