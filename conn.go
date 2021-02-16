package gn

import (
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cofepy/gn/common"
)

// Conn 客户端长连接
type Conn struct {
	s            *Server        // 服务器引用
	fd           int32          // 文件描述符
	addr         string         // 客户端地址
	buffer       *common.Buffer // 读缓存区
	lastReadTime time.Time      // 最后一次读取数据的时间
	data         interface{}    // 业务自定义数据，用作扩展
}

// newConn 创建到客户端的tcp连接
func newConn(fd int32, addr string, s *Server) *Conn {
	return &Conn{
		s:            s,
		fd:           fd,
		addr:         addr,
		buffer:       common.NewBuffer(s.readBufferPool.Get().([]byte)),
		lastReadTime: time.Now(),
	}
}

// GetFd 获取文件描述符
func (c *Conn) GetFd() int32 {
	return c.fd
}

// GetAddr 获取客户端地址
func (c *Conn) GetAddr() string {
	return c.addr
}

// Read 读取数据
func (c *Conn) Read() error {
	c.lastReadTime = time.Now()
	fd := int(c.GetFd())

	for {
		// 从缓存区读数据
		err := c.buffer.ReadFromFD(fd)

		if err != nil {
			// 缓存区暂无数据可读
			if err == syscall.EAGAIN {
				log.Info("无数据可读")
				return nil
			}
			log.Info("ReadFromFD出错")
			return err
		}

		err = c.s.decoder.Decode(c)
		if err != nil {
			log.Info("数据解析出错")
			return err
		}
	}

}

// Write 写入数据
func (c *Conn) Write(bytes []byte) (int, error) {
	return syscall.Write(int(c.fd), bytes)
}

// Close 关闭连接
func (c *Conn) Close() error {
	// 从epoll监听的文件描述符中删除
	err := c.s.poll.RemoveAndClose(int(c.fd))
	if err != nil {
		log.Error(err)
	}

	// 从conns中删除conn
	c.s.conns.Delete(c.fd)
	// 归还缓存区
	c.s.readBufferPool.Put(c.buffer.Buf)
	// 连接数减一
	atomic.AddInt64(&c.s.connsNum, -1)
	return nil
}

// GetData 获取数据
func (c *Conn) GetData() interface{} {
	return c.data
}

// SetData 设置数据
func (c *Conn) SetData(data interface{}) {
	c.data = data
}
