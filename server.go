package gn

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cofepy/gn/common"
	"github.com/cofepy/gn/poll"
)

var (
	MaxClient      = 10
	ErrReadTimeout = errors.New("tcp read timeout")
)

var log = common.GetLogger()

// Handler Server 注册接口
type Handler interface {
	OnConnect(c *Conn)               // OnConnect 当TCP长连接建立成功是回调
	OnMessage(c *Conn, bytes []byte) // OnMessage 当客户端有数据写入是回调
	OnClose(c *Conn, err error)      // OnClose 当客户端主动断开链接或者超时时回调,err返回关闭的原因
}

// server TCP服务
type Server struct {
	options  *options // 服务参数
	serverFd int      //服务端句柄
	stop     chan int // 服务器关闭信号

	poll    *poll.Poll // 多路复用
	handler Handler    // 注册的处理
	decoder Decoder    // 解码器

	readBufferPool *sync.Pool // 读缓存区池，从池子中可以取出缓存区，分配给创建的新连接（服务端与客户端用来读）

	EventQueueNum int32               // 事件队列的数量
	EventQueues   []chan common.Event // 事件队列

	conns    sync.Map // TCP长连接管理
	connsNum int64    // 当前建立的长连接数量

}

// NewServer 创建server服务器
func NewServer(port int, handler Handler, decoder Decoder, opts ...Option) (*Server, error) {
	options := getDefaultOptions(opts...)

	// 初始化读缓存区池
	readBufferPool := &sync.Pool{
		New: func() interface{} {
			b := make([]byte, options.readBufferLen)
			return b
		},
	}

	// 初始化poll网络
	poll, err := poll.NewPoll(port)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// 创建socket
	serverAddr := &syscall.SockaddrInet4{Port: port}
	serverFd, err := createListener(serverAddr)
	if err != nil {
		panic(err)
	}

	// 监听socket
	log.Info("\t将服务端socket添加到监听事件表:", serverFd)
	poll.AddRead(serverFd)

	// 初始化io事件队列
	EventQueues := make([]chan common.Event, options.eventGoNum)
	for i := range EventQueues {
		EventQueues[i] = make(chan common.Event, options.EventQueueLen)
	}

	// 创建服务器
	return &Server{
		options:        options,
		serverFd:       serverFd,
		poll:           poll,
		readBufferPool: readBufferPool,
		handler:        handler,
		decoder:        decoder,
		EventQueues:    EventQueues,
		EventQueueNum:  int32(options.eventGoNum),
		conns:          sync.Map{},
		connsNum:       0,
		stop:           make(chan int),
	}, nil
}

// Run 启动服务
func (s *Server) Run() {
	log.Info("gn server run")
	// 开始接收连接
	// s.startAccept()

	// 处理请求
	s.startEventConsumer()

	// 定时器，检查连接是否超时，清理过期连接
	//s.checkTimeout()

	// 生产请求
	s.startEventProducer()
}

// Run 启动服务
func (s *Server) Stop() {
	// 关闭Server的stop通道
	close(s.stop)
	for _, queue := range s.EventQueues {
		// 关闭处理队列
		close(queue)
	}
}

// eventGateway 转发事件（由eventProduce调用，将事件转发给eventConsumer处理）
func (s *Server) eventGateway(event common.Event) {
	//根据（句柄id % 队列个数）的余数，确定由哪个队列处理，这样会使事件处理比较均匀
	index := event.Fd % s.EventQueueNum

	log.Info("转发事件: 目标队列编号:", index)
	log.Info(event.Fmt("short"))
	// 将事件发送到EventQueues中，等EventConsumer处理
	s.EventQueues[index] <- event

}

// startAccept 开始接受连接请求
func (s *Server) startAccept() {
	for i := 0; i < s.options.acceptGoNum; i++ {
		go s.accept()
	}
	log.Info(fmt.Sprintf("start accept by %d goroutine", s.options.acceptGoNum))
}

// accept 接受连接请求
func (s *Server) accept() {
	for {
		select {
		case <-s.stop:
			return
		default:
			s.NewClientConn("accept")
		}
	}
}

func (s *Server) NewClientConn(caller string) {
	// clientFd是客户端的socket句柄，sockAddr是socket连接的地址
	// log.Info("等待连接请求, caller:", caller)

	// 从服务端当前的socket句柄，拿到客户端句柄
	// 该函数一直阻塞，直到有新的请求进来
	clientInt, sockAddr, err := syscall.Accept(s.serverFd)

	log.Info("=========> 新连接, serverFd:", s.serverFd, " clientFd:", clientInt)

	if err != nil {
		log.Info("===> error:", err)
		return
	}

	// 设置为非阻塞状态
	syscall.SetNonblock(clientInt, true)

	// 创建到客户端的连接
	clientFd := int32(clientInt)

	// 创建新连接
	conn := newConn(clientFd, getIPPort(sockAddr), s)

	// 将连接保存到conns表
	log.Info("将连接保存到表，客户端句柄:", clientFd)
	s.conns.Store(clientFd, conn)

	// 当前连接数加1
	atomic.AddInt64(&s.connsNum, 1)

	// 客户端连接事件
	s.handler.OnConnect(conn)

	// 将客户端读事件添加到poll中
	log.Info("为客户端添加读, 客户端句柄:", clientFd)

	//err = s.poll.AddReadWrite(clientInt)
	err = s.poll.AddRead(clientInt)

	if err != nil {
		log.Error(err)
		return
	}
	s.ShowConns()

}

// StartProducer 启动生产者
func (s *Server) startEventProducer() {
	log.Info("start event producer")
	for {
		time.Sleep(time.Second * 3)

		select {
		case <-s.stop:
			log.Error("stop producer")
			return
		default:
			log.Info("============> EventProducer")
			//log.Info(s.poll.ShowChanges())
			err := s.poll.Wait(s.eventGateway)
			//log.Info("对事件进行分发完成")
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func SockaddrToAddr(sa syscall.Sockaddr) (syscall.SockaddrInet4, error) {
	var addr syscall.SockaddrInet4
	switch sa := sa.(type) {
	case *syscall.SockaddrInet4:
		addr.Addr = sa.Addr
		addr.Port = sa.Port
		return addr, nil
	default:
		return addr, errors.New("currently unsupported socket address")
	}
}

// startEventConsumer 启动消费者, 消费s.EventQueues中的事件
func (s *Server) startEventConsumer() {
	for _, queue := range s.EventQueues {
		go s.consumeEvent(queue)
	}
	log.Info(fmt.Sprintf("start io event consumer by %d goroutine", len(s.EventQueues)))
}

// consumeEvent 消费事件
func (s *Server) consumeEvent(queue chan common.Event) {

	for event := range queue {

		log.Info("============> EventConsume")
		log.Info("事件句柄， fd:", event.Fd)
		// 从连接表中查找该事件句柄对应的连接
		c, ok := s.GetConn(event.Fd)
		log.Info("对应连接是否找到:", ok)
		if !ok {
			s.ShowConns()
		}

		switch {

		case event.Fd == int32(s.serverFd), ok == false:
			log.Info("switch 1 新的客户端连接")

			s.NewClientConn("consume")

		case event.Type == common.EventClose:
			log.Info("switch 2 关闭客户端连接")

			c.Close()
			s.handler.OnClose(c, io.EOF)

		case event.Type == common.EventTimeout:
			log.Info("switch 3 客户端连接超时")

			c.Close()
			s.handler.OnClose(c, ErrReadTimeout)
			s.ShowConns()

		default:
			log.Info("switch 4 收到客户端数据")

			err := c.Read()
			if err != nil {
				if err == syscall.EBADF {
					// 服务端关闭连接
					continue
				}
				log.Info("读取数据出错，关闭连接，error:", err)
				c.Close()
				s.handler.OnClose(c, err)
				log.Debug(err)
			}
		}
	}
}

// GetConnsNum 获取当前长连接的数量
func (s *Server) GetConnsNum() int64 {
	return atomic.LoadInt64(&s.connsNum)
}

func getIPPort(sa syscall.Sockaddr) string {
	addr := sa.(*syscall.SockaddrInet4)
	return fmt.Sprintf("%d.%d.%d.%d:%d", addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3], addr.Port)
}

// checkTimeout 定时检查超时的TCP长连接
func (s *Server) checkTimeout() {
	if s.options.timeout == 0 || s.options.timeoutTicker == 0 {
		return
	}
	log.Info(fmt.Sprintf("check timeout goroutine run,check_time:%v,timeout:%v", s.options.timeoutTicker, s.options.timeout))
	go func() {
		ticker := time.NewTicker(s.options.timeoutTicker)
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				s.conns.Range(func(key, value interface{}) bool {
					c := value.(*Conn)
					if time.Now().Sub(c.lastReadTime) > s.options.timeout {
						// 循环遍历所有连接，检查是否超时
						s.eventGateway(common.Event{Fd: c.fd, Type: common.EventTimeout})
					}
					return true
				})
			}
		}
	}()
}

func createListener(serverAddr *syscall.SockaddrInet4) (fd int, err error) {
	serverFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return serverFd, err
	}
	err = syscall.Bind(serverFd, serverAddr)
	if err != nil {
		return serverFd, err
	}
	err = syscall.Listen(serverFd, MaxClient)
	if err != nil {
		return serverFd, err
	}
	fmt.Printf("initially create socket fd listening: %s:%d \n", strings.Replace(strings.Trim(fmt.Sprint(serverAddr.Addr), "[]"), " ", ".", -1), serverAddr.Port)
	return serverFd, nil
}

// GetConn 获取Conn
func (s *Server) GetConn(fd int32) (*Conn, bool) {
	value, ok := s.conns.Load(fd)
	if !ok {
		return nil, false
	}
	return value.(*Conn), true
}

func (s *Server) ShowConns() {
	log.Info("当前所有连接：")
	log.Info("\t------------------")
	s.conns.Range(func(key, value interface{}) bool {
		log.Info("\tkey:", key, " value:", value.(*Conn).addr)
		return true
	})
	log.Info("\t------------------")
}
