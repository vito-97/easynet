package easynet

import (
	"reflect"
	"runtime"
)

func nameOfFunction(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
