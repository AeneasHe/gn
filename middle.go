package gn

import (
	"time"
)

// WithReadBufferLen 设置缓存区大小
func WithReadBufferLen(len int) Option {
	return newFuncServerOption(func(o *options) {
		if len <= 0 {
			panic("acceptGoNum must greater than 0")
		}
		o.readBufferLen = len
	})
}

// WithAcceptGNum 设置建立连接的goroutine数量
func WithAcceptGNum(num int) Option {
	return newFuncServerOption(func(o *options) {
		if num <= 0 {
			panic("acceptGoNum must greater than 0")
		}
		o.acceptGoNum = num
	})
}

// WithIOGNum 设置处理IO的goroutine数量
func WithIOGNum(num int) Option {
	return newFuncServerOption(func(o *options) {
		if num <= 0 {
			panic("IOGNum must greater than 0")
		}
		o.eventGoNum = num
	})
}

// WithIOEventQueueLen 设置IO事件队列长度，默认值是1024
func WithIOEventQueueLen(num int) Option {
	return newFuncServerOption(func(o *options) {
		if num <= 0 {
			panic("EventQueueLen must greater than 0")
		}
		o.EventQueueLen = num
	})
}

// WithTimeout 设置TCP超时检查的间隔时间以及超时时间
func WithTimeout(timeoutTicker, timeout time.Duration) Option {
	return newFuncServerOption(func(o *options) {
		if timeoutTicker <= 0 {
			panic("timeoutTicker must greater than 0")
		}
		if timeout <= 0 {
			panic("timeoutTicker must greater than 0")
		}

		o.timeoutTicker = timeoutTicker
		o.timeout = timeout
	})
}
