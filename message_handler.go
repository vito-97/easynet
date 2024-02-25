package easynet

type HandlerFunc func(req IRequest)

type HandlersChain []HandlerFunc

// Last 返回最后一个处理函数，最后一个是处理主要函数
func (c HandlersChain) Last() HandlerFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}
