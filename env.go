package easynet

import (
	"os"
	"path/filepath"
)

const (
	// EnvConfigFilePathKey
	// (设置配置文件路径 export EASYNET_CONFIG_FILE = xxx/xxx/easynet.json)
	EnvConfigFilePathKey     = "EASYNET_CONFIG_FILE"
	EnvDefaultConfigFilePath = "/config/easynet.json"
)

var env = new(environment)

type environment struct {
	configFilePath string
	initialized    bool
}

func init() {
	env.init()
}

func (e *environment) init() {
	if e.initialized {
		return
	}
	e.initialized = true
	configFilePath := os.Getenv(EnvConfigFilePathKey)
	if configFilePath == "" {
		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		configFilePath = filepath.Join(pwd, EnvDefaultConfigFilePath)
	}
	var err error
	configFilePath, err = filepath.Abs(configFilePath)
	if err != nil {
		panic(err)
	}
	env.configFilePath = configFilePath
}

func GetConfigFilePath() string {
	return env.configFilePath
}
