package database

import (
	"context"
	"sync"

	"git2.qingtingfm.com/infra/qt-boot/pkg/log"
)

// 模型定义的内存缓存

var modelDefCache = make(map[string]*ModelDef)
var registerModelMu sync.Mutex

// 注册dao模型
// 此方法需要在dao文件的init()方法中调用
func RegisterModel(m Model) {
	registerModelMu.Lock() // 并发保护，以防万一
	defer registerModelMu.Unlock()

	fullName, _ := GetModelFullNameAndValue(m)

	if _, ok := modelDefCache[fullName]; ok {
		return
	}

	modelDef := NewModelDef(m)

	modelDefCache[fullName] = modelDef

	log.Infoc(context.Background(), "注册model: %s", fullName)
}
