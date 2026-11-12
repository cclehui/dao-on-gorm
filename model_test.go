package daoongorm

import (
	"reflect"
	"testing"
	"time"
)

func TestNewFieldTagParsesOptions(t *testing.T) {
	cases := []struct {
		tag          string
		wantColumn   string
		wantPK       bool
		wantIgnore   bool
		wantDefault  string
		wantHasDefau bool
	}{
		{tag: "", wantIgnore: true},
		{tag: "-", wantIgnore: true},
		{tag: "column:id;primaryKey", wantColumn: "id", wantPK: true},
		{tag: "column:weight;column_default:1.9", wantColumn: "weight", wantDefault: "1.9", wantHasDefau: true},
		{tag: "column:weight;default:1.9", wantColumn: "weight", wantDefault: "1.9", wantHasDefau: true},
		{tag: "column:a:b", wantColumn: "a:b"},
	}

	for _, c := range cases {
		got := NewFieldTag(c.tag)

		if got.ColumnName != c.wantColumn {
			t.Errorf("tag %q column = %q, want %q", c.tag, got.ColumnName, c.wantColumn)
		}
		if got.IsPK != c.wantPK {
			t.Errorf("tag %q IsPK = %v, want %v", c.tag, got.IsPK, c.wantPK)
		}
		if got.Ignore != c.wantIgnore {
			t.Errorf("tag %q Ignore = %v, want %v", c.tag, got.Ignore, c.wantIgnore)
		}
		if got.ColumnDefault != c.wantDefault || got.HaveColumnDefault != c.wantHasDefau {
			t.Errorf("tag %q default = (%q,%v), want (%q,%v)", c.tag,
				got.ColumnDefault, got.HaveColumnDefault, c.wantDefault, c.wantHasDefau)
		}
	}
}

type validModel struct {
	ID        int       `gorm:"column:id;primaryKey"`
	Title     string    `gorm:"column:title"`
	CreatedAt time.Time `gorm:"column:created_at"`
	daoBase   *DaoBase
}

func TestNewModelDefSkipsUnexportedAndSortsPK(t *testing.T) {
	modelDef := NewModelDef(&validModel{})

	if _, ok := modelDef.Fields["daoBase"]; ok {
		t.Error("unexported field should not be mapped")
	}

	if len(modelDef.PKFields) != 1 || modelDef.PKFieldNames[0] != "ID" {
		t.Fatalf("PK fields = %v, want [ID]", modelDef.PKFieldNames)
	}
}

type compositeModel struct {
	UserID   int `gorm:"column:user_id;primaryKey"`
	ColumnID int `gorm:"column:column_id;primaryKey"`
}

func TestNewModelDefSortsCompositePK(t *testing.T) {
	modelDef := NewModelDef(&compositeModel{})

	want := []string{"ColumnID", "UserID"}
	if !reflect.DeepEqual(modelDef.PKFieldNames, want) {
		t.Fatalf("PKFieldNames = %v, want %v", modelDef.PKFieldNames, want)
	}
}

type noPKModel struct {
	Title string `gorm:"column:title"`
}

func TestNewModelDefRequiresPrimaryKey(t *testing.T) {
	requirePanic(t, func() { NewModelDef(&noPKModel{}) })
}

type unsupportedModel struct {
	ID     int               `gorm:"column:id;primaryKey"`
	Extras map[string]string `gorm:"column:extras"`
}

func TestNewModelDefRejectsUnsupportedFieldType(t *testing.T) {
	requirePanic(t, func() { NewModelDef(&unsupportedModel{}) })
}

type duplicateColumnModel struct {
	ID    int    `gorm:"column:id;primaryKey"`
	Title string `gorm:"column:id"`
}

func TestNewModelDefRejectsDuplicateColumn(t *testing.T) {
	requirePanic(t, func() { NewModelDef(&duplicateColumnModel{}) })
}

func TestGetModelFullNameAndValueRejectsBadInput(t *testing.T) {
	requirePanic(t, func() { GetModelFullNameAndValue(nil) })
	requirePanic(t, func() { GetModelFullNameAndValue(validModel{}) })
	requirePanic(t, func() { GetModelFullNameAndValue((*validModel)(nil)) })
}

func TestJoinPKValuesIsStableAndUnambiguous(t *testing.T) {
	if got, want := JoinPKValues([]string{"b", "a"}), JoinPKValues([]string{"a", "b"}); got != want {
		t.Fatalf("JoinPKValues should be order independent, got %q and %q", got, want)
	}

	if JoinPKValues([]string{"a", "bc"}) == JoinPKValues([]string{"ab", "c"}) {
		t.Fatal("JoinPKValues should not collide for ambiguous splits")
	}
}

func requirePanic(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic")
		}
	}()

	fn()
}
