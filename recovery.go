package easynet

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"
)

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)

type RecoveryFunc func(c IRequest, err any)

// Recovery 返回一个捕获recovery中间件
func Recovery() HandlerFunc {
	return RecoveryWithWriter(DefaultErrorWriter)
}

// CustomRecovery 返回一个捕获recovery中间件，并且设置一个回调函数处理
func CustomRecovery(handle RecoveryFunc) HandlerFunc {
	return RecoveryWithWriter(DefaultErrorWriter, handle)
}

// RecoveryWithWriter  返回一个捕获recovery中间件，并且将错误详情输出到writer中，可接收一个回调函数处理
func RecoveryWithWriter(out io.Writer, recovery ...RecoveryFunc) HandlerFunc {
	if len(recovery) > 0 {
		return CustomRecoveryWithWriter(out, recovery[0])
	}
	return CustomRecoveryWithWriter(out, defaultHandleRecovery)
}

// CustomRecoveryWithWriter 返回一个捕获recovery中间件，并且将错误详情输出到writer中，接收一个回调函数处理
func CustomRecoveryWithWriter(out io.Writer, handle RecoveryFunc) HandlerFunc {
	var logger *log.Logger

	if out != nil {
		logger = log.New(out, "\n\n\x1b[31m", log.LstdFlags)
	}

	return func(req IRequest) {
		defer func() {
			if err := recover(); err != nil {
				if logger != nil {
					s := stack(3)

					data := req.Data()

					logger.Printf("[Recovery] %s panic recovered:\n%s\n%s\n%s%s",
						timeFormat(time.Now()), data, err, s, reset)
				}

				handle(req, err)
			}
		}()

		req.Next()
	}
}

func defaultHandleRecovery(req IRequest, _ any) {
	req.Abort()
}

// stack 获取调用栈的信息，会跳过指定的层数
func stack(ship int) []byte {
	buf := bytes.NewBuffer(nil)

	var lines [][]byte
	var lastFile string

	for i := ship; ; i++ {
		pc, file, line, ok := runtime.Caller(i)

		if !ok {
			break
		}

		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}

	return buf.Bytes()
}

// source 返回第N行的内容
func source(lines [][]byte, n int) []byte {
	//在堆栈信息中，行是从1开始的
	n--
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// function 返回pc指针的函数名称
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)

	if fn == nil {
		return dunno
	}

	name := []byte(fn.Name())

	// runtime/debug.*T·ptrMethod
	// 消除/分割再消除点前面的，再将中心点替换为点
	// *T.ptrMethod
	if lastSlash := bytes.LastIndex(name, slash); lastSlash > 0 {
		name = name[lastSlash+1:]
	}

	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.ReplaceAll(name, centerDot, dot)
	return name
}

func timeFormat(t time.Time) string {
	return t.Format("2006/01/02 - 15:04:05")
}
