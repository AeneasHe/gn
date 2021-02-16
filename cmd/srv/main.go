package main

import (
	"bytes"
	"os"
	"syscall"
	"time"

	"os/signal"

	"github.com/cofepy/gn"
	"github.com/cofepy/gn/common"
)

var log = common.GetLogger()
var headerLen = 2
var port = 8080

var server *gn.Server

var encoder = gn.NewHeaderLenEncoder(2, 1024)

type Handler struct {
}

func BytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}
func (Handler) OnConnect(c *gn.Conn) {
	log.Info("\tconnect:", c.GetFd(), c.GetAddr())
}

func (Handler) OnMessage(c *gn.Conn, data []byte) {
	log.Info("\t客户端新数据:【", string(data), "】")

	response := BytesCombine([]byte("你好："), data)
	encoder.EncodeToFD(c.GetFd(), response)
	log.Info("\t发送响应数据:【", string(response), "】")

}

func (Handler) OnClose(c *gn.Conn, err error) {
	log.Info("close:", c.GetFd(), err)
}

func main() {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan)

	var err error

	server, err = gn.NewServer(
		port,
		&Handler{},
		gn.NewHeaderLenDecoder(headerLen),
		gn.WithTimeout(1*time.Second, 5*time.Second),
		gn.WithReadBufferLen(10))

	if err != nil {
		log.Info("err")
		return
	}

	go server.Run()

	for {
		select {
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGQUIT, os.Interrupt:
				log.Info("接受到退出信号：", sig, " 正在退出...")
				server.Stop()
				time.Sleep(time.Second * 3)
				os.Exit(0)
			}
		}
	}
}
