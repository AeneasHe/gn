package common

import "fmt"

const (
	EventIn      = 1 // 数据流入
	EventClose   = 2 // 断开连接
	EventTimeout = 3 // 检测到超时
)

type Event struct {
	Fd   int32 // 文件描述符
	Type int32 // 事件类型
}

func (e *Event) Fmt(flag string) string {
	var s string
	if flag == "long" {
		s += fmt.Sprintf("事件信息:【 句柄:%d", e.Fd)
		switch {
		case e.Type == EventIn:
			s += " 类型: 数据流入"
		case e.Type == EventClose:
			s += " 类型: 断开连接"
		case e.Type == EventTimeout:
			s += " 类型: 连接超时"
		}
		s += "】"
	}
	if flag == "short" {
		s += "事件"
		switch {
		case e.Type == EventIn:
			s += "类型: 数据流入"
		case e.Type == EventClose:
			s += "类型: 断开连接"
		case e.Type == EventTimeout:
			s += "类型: 连接超时"
		}
	}
	return s

}
