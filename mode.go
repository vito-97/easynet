package easynet

import "os"

type EnvMode string

const EnvModeKey = "EASY_NET_MODE"

const (
	// DebugMode debug模式
	DebugMode EnvMode = "debug"
	// ReleaseMode 发行模式
	ReleaseMode EnvMode = "release"
	// TestMode 测试模式
	TestMode EnvMode = "test"
)

var (
	modeName = DebugMode
)

func init() {
	mode := EnvMode(os.Getenv(EnvModeKey))
	SetMode(mode)
}

func SetMode(value EnvMode) {
	if value == "" {
		value = DebugMode
	}

	switch value {
	case DebugMode:
	case ReleaseMode:
	case TestMode:
	default:
		panic("gin mode unknown: " + value + " (available mode: debug release test)")
	}

	modeName = value
}

func Mode() EnvMode {
	return modeName
}
