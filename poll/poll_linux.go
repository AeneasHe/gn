// +build linux

package poll

import (
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/cofepy/gn/common"
)

var log = common.GetLogger()

// 对端关闭连接 8193
const (
	EpollRead  = syscall.EPOLLIN | syscall.EPOLLPRI | syscall.EPOLLERR | syscall.EPOLLHUP | unix.EPOLLET | syscall.EPOLLRDHUP
	EpollClose = uint32(syscall.EPOLLIN | syscall.EPOLLRDHUP)
)

type Poll struct {
	fd int
}

// 创建Poll
func NewPoll(port int) (*Poll, error) {

	fd, err := syscall.NewPoll1(0)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &Poll{
		fd: fd,
	}, nil
}

func (p *Poll) AddRead(fd int) error {
	err := syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Events: EpollRead,
		Fd:     int32(fd),
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *Poll) RemoveAndClose(fd int) error {
	// 移除文件描述符的监听
	err := syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}

	// 关闭文件描述符
	err = syscall.Close(fd)
	if err != nil {
		return err
	}

	return nil
}

func (p *Poll) Wait(eventGateway func(event common.Event)) error {
	events := make([]syscall.EpollEvent, 100)
	n, err := syscall.Wait(p.fd, events, -1)
	if err != nil {
		return err
	}

	for i := 0; i < n; i++ {
		if events[i].Events == EpollClose {
			eventGateway(Event{
				Fd:   events[i].Fd,
				Type: common.EventClose,
			})
		} else {
			eventGateway(Event{
				Fd:   events[i].Fd,
				Type: common.EventIn,
			})
		}
	}

	return nil
}

func (p *Poll) Accept() (int, syscall.Sockaddr, error) {

	nfd, sa, err := syscall.Accept(int(p.fd))
	if err != nil {
		return 0, nil, nil

	}
	return nfd, sa, nil

}
