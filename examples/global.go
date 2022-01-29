package dao

var configFile = "./config_demo.yaml"

var configDemo *ConfigDemo
var dbClient *DBClientDemo
var cacheUtil *CacheUtilDemo

// db client
func GetDBClient() *DBClientDemo {
	if dbClient == nil {
		initBase()
	}

	return dbClient
}

// 缓存组件
func GetCacheUtil() *CacheUtilDemo {
	if cacheUtil == nil {
		initBase()
	}

	return cacheUtil
}

func initBase() {
	initConfig()
	initCacheUtil()
	initDBClient()
}

func initConfig() {
	configDemoTmp := &ConfigDemo{}

	_, err := configDemoTmp.DecodeFromFile(configFile)
	if err != nil {
		panic(err)
	}

	configDemo = configDemoTmp
}

func initDBClient() {
	dbClientTmp, err := NewDBClientDemo(configDemo.Mysql.Test)
	if err != nil {
		panic(err)
	}

	dbClient = dbClientTmp
}

func initCacheUtil() {
	cacheUtilTmp := NewCacheUtilDemo(configDemo.Redis.Default)

	cacheUtil = cacheUtilTmp
}
