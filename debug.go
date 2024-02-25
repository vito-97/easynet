package easynet

import (
	"fmt"
	"strings"
)

func IsDebugging() bool {
	return DebugMode == modeName
}

// DebugPrintRouteFunc 调试日志输出格式
var DebugPrintRouteFunc func(t uint32, handlerName string, numHandlers int)

func debugPrintRoute(t uint32, handlers HandlersChain) {
	if IsDebugging() {
		numHandlers := len(handlers)
		handlerName := nameOfFunction(handlers.Last())
		if DebugPrintRouteFunc == nil {
			debugPrint("route id %d --> %s (%d handlers)\n", t, handlerName, numHandlers)
		} else {
			DebugPrintRouteFunc(t, handlerName, numHandlers)
		}
	}
}

// debugPrint 最终调用fmt.Fprintf输出到writer中
func debugPrint(format string, values ...any) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		fmt.Fprintf(DefaultWriter, "[EasyNet-debug] "+format, values...)
	}
}

func debugPrintError(err error) {
	if err != nil && IsDebugging() {
		fmt.Fprintf(DefaultErrorWriter, "[EasyNet-debug] [ERROR] %v\n", err)
	}
}
