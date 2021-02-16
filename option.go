package gn

import (
	"runtime"
	"time"
)

// options Server初始化参数
type options struct {
	readBufferLen int           // 所读取的客户端包的最大长度，客户端发送的包不能超过这个长度，默认值是1024字节
	acceptGoNum   int           // 处理初始连接的goroutine数量
	eventGoNum    int           // 处理事件的goroutine数量
	EventQueueLen int           // 事件队列的数量
	timeoutTicker time.Duration // 超时时间检查间隔
	timeout       time.Duration // 超时时间
}

type Option interface {
	apply(*options)
}

type funcServerOption struct {
	f func(*options)
}

func (fdo *funcServerOption) apply(do *options) {
	fdo.f(do)
}

func newFuncServerOption(f func(*options)) *funcServerOption {
	return &funcServerOption{
		f: f,
	}
}

func getDefaultOptions(opts ...Option) *options {
	cpuNum := runtime.NumCPU()
	options := &options{
		readBufferLen: 1024,
		acceptGoNum:   cpuNum,
		eventGoNum:    cpuNum,
		EventQueueLen: 1024,
	}

	for _, o := range opts {
		o.apply(options)
	}
	return options
}
