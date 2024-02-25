package easynet

import "os"

const EnvMode = "EASY_NET_MODE"

const (
	// DebugMode debug模式
	DebugMode = "debug"
	// ReleaseMode 发行模式
	ReleaseMode = "release"
	// TestMode 测试模式
	TestMode = "test"
)

var (
	modeName = DebugMode
)

func init() {
	mode := os.Getenv(EnvMode)
	SetMode(mode)
}

func SetMode(value string) {
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

func Mode() string {
	return modeName
}
