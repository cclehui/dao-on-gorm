package daoongorm

import (
	"context"
	"testing"
	"time"
)

type diffModel struct {
	ID        int       `gorm:"column:id;primaryKey"`
	Version   int64     `gorm:"column:version"`
	Title     string    `gorm:"column:title"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`

	daoBase *DaoBase
}

func (m *diffModel) TableName() string { return "diff_model" }
func (m *diffModel) DBName() string    { return "test_db" }
func (m *diffModel) DBClient() DBClientInterface {
	return nil
}
func (m *diffModel) GetDaoBase() *DaoBase     { return m.daoBase }
func (m *diffModel) SetDaoBase(base *DaoBase) { m.daoBase = base }

// newTestDao 构造一个不连库的 DaoBase, 用于验证纯内存逻辑.
func newTestDao(t *testing.T, model *diffModel) *DaoBase {
	t.Helper()

	RegisterModel(model)

	daoBase, err := NewDaoBaseNoLoad(model)
	if err != nil {
		t.Fatalf("NewDaoBaseNoLoad: %v", err)
	}

	model.SetDaoBase(daoBase)

	return daoBase
}

// 只有真正变化的列才会出现在 UPDATE 里: 全列更新会覆盖并发写入.
func TestChangedColumnValues(t *testing.T) {
	now := time.Now()
	model := &diffModel{ID: 1, Version: 2, Title: "a", CreatedAt: now, UpdatedAt: now}
	daoBase := newTestDao(t, model)

	if changed := daoBase.changedColumnValues(); len(changed) != 0 {
		t.Fatalf("unchanged row should produce no updates, got %v", changed)
	}

	model.Title = "b"

	changed := daoBase.changedColumnValues()
	if len(changed) != 1 || changed["title"] != "b" {
		t.Fatalf("changed = %v, want only title", changed)
	}
}

// time.Time 带 loc 指针, 用 == 比较会误判成变化; DeepEqual 不会.
func TestChangedColumnValuesIgnoresEqualTime(t *testing.T) {
	ts := time.Now()
	model := &diffModel{ID: 1, CreatedAt: ts, UpdatedAt: ts}
	daoBase := newTestDao(t, model)

	// 同一时刻, 不同 Location: DeepEqual 会判为不等, Equal 才是正确语义.
	model.UpdatedAt = ts.In(time.FixedZone("CST", 8*3600))

	if changed := daoBase.changedColumnValues(); len(changed) != 0 {
		t.Fatalf("equal timestamps should not count as change, got %v", changed)
	}
}

// Save 会把 updated_at touch 到当前时间, 但不会动调用方显式赋的值.
func TestUpdateAutoFillTouchSemantics(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	model := &diffModel{ID: 1, Title: "a", CreatedAt: old, UpdatedAt: old}
	daoBase := newTestDao(t, model)

	daoBase.updateAutoFillValue()
	if !model.UpdatedAt.After(old) {
		t.Fatal("updated_at should be touched when the caller left it unchanged")
	}

	explicit := time.Date(2021, 5, 6, 7, 8, 9, 0, time.UTC)
	model.UpdatedAt = explicit
	daoBase.updateAutoFillValue()

	if !model.UpdatedAt.Equal(explicit) {
		t.Errorf("updated_at = %v, want the explicit value to be kept", model.UpdatedAt)
	}
}

func TestCreateAutoFillKeepsExplicitValues(t *testing.T) {
	explicit := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	model := &diffModel{CreatedAt: explicit}
	daoBase := newTestDao(t, model)

	daoBase.createAutoFillValue()

	if !model.CreatedAt.Equal(explicit) {
		t.Errorf("created_at = %v, want the explicit value to be kept", model.CreatedAt)
	}
	if model.UpdatedAt.IsZero() {
		t.Error("updated_at should be filled when zero")
	}
}

// precision 是列属性: 声明了 precision:3 才舍入, 没声明原样交给 gorm.
func TestRoundToFieldPrecision(t *testing.T) {
	value := time.Date(2026, 11, 12, 10, 20, 30, 123456789, time.UTC)

	cases := []struct {
		precision int
		want      time.Time
	}{
		{0, value},
		{-1, value},
		{3, time.Date(2026, 11, 12, 10, 20, 30, 123000000, time.UTC)},
		{6, time.Date(2026, 11, 12, 10, 20, 30, 123457000, time.UTC)},
		{42, value},
	}

	for _, c := range cases {
		if got := roundToFieldPrecision(value, c.precision); !got.Equal(c.want) {
			t.Errorf("precision %d: got %v, want %v", c.precision, got, c.want)
		}
	}
}

// 显式赋值也要归一化, 否则内存值和库里不一致.
func TestRoundTimeFieldsNormalizesExplicitValues(t *testing.T) {
	model := &precisionModel{
		CreatedAt: time.Date(2026, 11, 12, 10, 20, 30, 123_600_000, time.UTC),
		UpdatedAt: time.Date(2026, 11, 12, 10, 20, 30, 123_456_789, time.UTC),
	}
	daoBase := newTestDaoForPrecision(t, model)

	daoBase.roundTimeFields()

	if got := model.CreatedAt.Nanosecond(); got != 124_000_000 {
		t.Errorf("precision:3 created_at = %d ns, want 124000000 (rounded up like MySQL)", got)
	}
	if got := model.UpdatedAt.Nanosecond(); got != 123_457_000 {
		t.Errorf("precision:6 updated_at = %d ns, want 123457000", got)
	}
}

// 没有 precision 声明的字段, created_at/updated_at 不会被处理.
func TestAutoFillRespectsFieldPrecision(t *testing.T) {
	model := &diffModel{}
	daoBase := newTestDao(t, model)
	daoBase.createAutoFillValue()

	if model.CreatedAt.Nanosecond() == 0 {
		t.Error("no-precision field should keep sub-second digits")
	}

	preciseModel := &precisionModel{}
	preciseBase := newTestDaoForPrecision(t, preciseModel)
	preciseBase.createAutoFillValue()

	if preciseModel.CreatedAt.Nanosecond()%1_000_000 != 0 {
		t.Errorf("precision:3 field should be rounded to ms, got %v", preciseModel.CreatedAt)
	}
	if preciseModel.UpdatedAt.Nanosecond()%1_000 != 0 {
		t.Errorf("precision:6 field should be rounded to us, got %v", preciseModel.UpdatedAt)
	}
}

// precisionModel 声明了 gorm 原生 precision, 用于验证按列精度截断.
type precisionModel struct {
	ID        int       `gorm:"column:id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at;precision:3"`
	UpdatedAt time.Time `gorm:"column:updated_at;precision:6"`
}

func (m *precisionModel) TableName() string           { return "precision_model" }
func (m *precisionModel) DBName() string              { return "test_db" }
func (m *precisionModel) DBClient() DBClientInterface { return nil }
func (m *precisionModel) GetDaoBase() *DaoBase        { return nil }
func (m *precisionModel) SetDaoBase(base *DaoBase)    {}

func newTestDaoForPrecision(t *testing.T, model *precisionModel) *DaoBase {
	t.Helper()

	RegisterModel(model)

	daoBase, err := NewDaoBaseNoLoad(model)
	if err != nil {
		t.Fatalf("NewDaoBaseNoLoad: %v", err)
	}

	return daoBase
}

func TestSetFieldDefaultValue(t *testing.T) {
	model := &diffModel{}
	daoBase := newTestDao(t, model)

	daoBase.setFieldDefaultValue()

	if model.ID != 0 || model.Version != 0 || model.Title != "" {
		t.Fatalf("defaults not applied: %+v", model)
	}
}

func TestDeleteCacheIsNoopWhenCacheDisabled(t *testing.T) {
	model := &diffModel{}
	daoBase := newTestDao(t, model)

	// NopCacheUtil 下调用不应 panic; 这里主要验证 option 生效路径.
	daoBase.DeleteCache(context.Background())
}
