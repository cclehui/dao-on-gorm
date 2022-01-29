package dao

import daoongorm "github.com/cclehui/dao-on-gorm"

// 把模型注册到gaoongorm中去， 方便全局管理
func init() {
	daoongorm.RegisterModel(&CclehuiTestADao{})
	daoongorm.RegisterModel(&CclehuiTestBDao{})
	daoongorm.RegisterModel(&CclehuiTestCDao{})
	daoongorm.RegisterModel(&CclehuiTestDDao{})
}
