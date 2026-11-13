package dao

// 模型注册在 model package 的 init 里, 这里 import 一次保证它被执行.
import (
	_ "github.com/cclehui/dao-on-gorm/examples/model"
)
