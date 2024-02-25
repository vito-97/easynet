package easynet

import (
	"fmt"
	"io"
	"time"
)

const (
	reset = "\033[0m"
)

type Skipper func(req IRequest) bool
type LogFormatter func(params LogFormatterParams) string

type LogFormatterParams struct {
	Request IRequest
	Type    uint32

	ClientIP string

	TimeStamp time.Time
	Latency   time.Duration
}

type LogConfig struct {
	Formatter LogFormatter

	Output io.Writer

	SkipTypes []uint32
	Skip      Skipper
}

func Logger() HandlerFunc {
	return LoggerWithConfig(LogConfig{})
}

func LoggerWithFormatter(f LogFormatter) HandlerFunc {
	return LoggerWithConfig(LogConfig{
		Formatter: f,
	})
}

func LoggerWithWriter(out io.Writer, notLogged ...uint32) HandlerFunc {
	return LoggerWithConfig(LogConfig{
		Output:    out,
		SkipTypes: notLogged,
	})
}

func LoggerWithConfig(config LogConfig) HandlerFunc {
	formatter := config.Formatter

	if formatter == nil {
		formatter = defaultLogFormatter
	}

	output := config.Output

	if output == nil {
		output = DefaultWriter
	}

	skipTypes := config.SkipTypes

	var skip map[uint32]struct{}

	if length := len(skipTypes); length > 0 {
		skip = make(map[uint32]struct{}, length)

		for _, t := range skipTypes {
			skip[t] = struct{}{}
		}
	}

	return func(req IRequest) {
		start := time.Now()

		req.Next()

		if _, ok := skip[req.GetMsgType()]; ok || (config.Skip != nil && config.Skip(req)) {
			return
		}

		params := LogFormatterParams{
			Request: req,
			Type:    req.GetMsgType(),
		}

		params.TimeStamp = time.Now()
		params.Latency = params.TimeStamp.Sub(start)
		params.ClientIP = req.GetConnection().GetRemoteAddrString()

		fmt.Fprint(output, formatter(params))
	}
}

func defaultLogFormatter(params LogFormatterParams) string {
	return fmt.Sprintf(
		"[EasyNet] %v | %10v | %15s | type %d\n%s",
		params.TimeStamp.Format("2006-01-02 15:04:05"),
		params.Latency,
		params.ClientIP,
		params.Type,
		params.Request.GetData(),
	)
}
