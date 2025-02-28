package easynet

import (
	"encoding/json"
	"os"
)

type Config struct {
	Name       string // 当前服务器名称
	ClientName string // 当前客户端名称
	Host       string // 当前服务器主机IP
	Port       int    // 当前服务器主机监听端口号

	Version          string     // 当前版本号
	MaxPacketSize    uint32     // 读写数据包的最大值
	MaxConn          int        // 当前服务器主机允许的最大连接个数
	WorkerPoolSize   uint32     // 业务工作Worker池的数量
	MaxWorkerTaskLen uint32     // 业务工作Worker对应负责的任务队列最大任务存储数量
	WorkerMode       WorkerMode // 为连接分配worker的方式
	MaxMsgChanLen    uint32     // 发送消息的缓冲最大长度
	IOReadBuffSize   uint32     // 每次IO最大的读取长度
}

func (c *Config) Reload() {
	configFilePath := GetConfigFilePath()
	if confFileExists, _ := pathExists(configFilePath); !confFileExists {
		debugPrint("config file %s is not exist! \n You can set config file by setting the environment variable %s, like export %s = xxx/xxx/easynet.conf\n", configFilePath, EnvConfigFilePathKey, EnvConfigFilePathKey)
		return
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, c)
	if err != nil {
		panic(err)
	}
}

var GlobalConfig *Config

func init() {
	GlobalConfig = &Config{
		Name:       "EasyNet Server",
		ClientName: "EasyNet Client",
		Host:       "0.0.0.0",
		Port:       8899,

		Version: "v1.0",

		MaxConn:          12000,
		MaxPacketSize:    4096,
		WorkerPoolSize:   10,
		MaxWorkerTaskLen: 1024,
		WorkerMode:       "",
		MaxMsgChanLen:    1024,
		IOReadBuffSize:   1024,
	}
	env.init()
	GlobalConfig.Reload()
}
