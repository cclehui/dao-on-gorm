package dao

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type ConfigDemo struct {
	Mysql *Mysql `yaml:"mysql"`
	Redis *Redis `yaml:"redis"`
}

func (c *ConfigDemo) DecodeFromFile(filePath string) (*ConfigDemo, error) {
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return c, err
	}

	err = yaml.Unmarshal(fileData, c)
	if err != nil {
		return c, errors.WithStack(err)
	}

	return c, nil
}

type Mysql struct {
	Test *DBClientConfig `yaml:"test"`
}

type Redis struct {
	Default *RedisConfig `yaml:"default"`
}
