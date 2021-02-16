// +build darwin

package poll

import (
	"fmt"
	"syscall"

	"github.com/cofepy/gn/common"
)

var log = common.GetLogger()

type Poll struct {
	fd      int
	changes []syscall.Kevent_t //事件列表

}

func NewPoll(port int) (*Poll, error) {

	poll := new(Poll)

	//获取kqueue队列操作具柄,并绑定到Poll
	kq, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}
	poll.fd = kq

	// 监听用户自定义事件
	_, err = syscall.Kevent(poll.fd, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		return nil, err
	}

	return poll, nil
}

func (p *Poll) AddRead(fd int) error {
	// log.Info("===> 1 add read，fd:", fd)
	// 添加“读事件“监听
	p.changes = append(p.changes,
		syscall.Kevent_t{
			Ident:  uint64(fd),
			Filter: syscall.EVFILT_READ,
			Flags:  syscall.EV_ADD,
		},
	)
	return nil
}

// AddReadWrite ...
func (p *Poll) AddReadWrite(fd int) error {
	// log.Info("===> 1 add read & write，fd:", fd)
	// 添加“读写事件“监听
	p.changes = append(p.changes,
		syscall.Kevent_t{
			Ident: uint64(fd), Filter: syscall.EVFILT_READ, Flags: syscall.EV_ADD,
		},
		syscall.Kevent_t{
			Ident: uint64(fd), Filter: syscall.EVFILT_WRITE, Flags: syscall.EV_ADD,
		},
	)
	log.Info("changes:", p.changes)
	return nil
}

// 监听事件，并处理
func (p *Poll) Wait(eventGateway func(event common.Event)) error {

	// 初始化：放处理结果的容器
	events := make([]syscall.Kevent_t, 128)

	// 监听p.changes，将结果放入events,返回值n是已经就绪的事件数量
	n, err := syscall.Kevent(p.fd, p.changes, events, nil)

	if err != nil && err != syscall.EINTR {
		return err
	}
	// log.Info("事件就绪数量, n:", n)
	// log.Info("changes:", p.changes)

	// p.changes = p.changes[:0]

	//循环调用eventGateway对events进行处理
	for i := 0; i < n; i++ {
		// log.Info("当前处理事件编号, i:", i)
		fd := int(events[i].Ident)
		if fd != 0 {
			_event := events[i]

			if _event.Filter == syscall.EVFILT_READ {
				log.Info("\t事件信息: flags:", _event.Flags, " filter:可以读")
			}
			if _event.Filter == syscall.EVFILT_WRITE {
				log.Info("\t事件信息: flags:", _event.Flags, " filter:可以写")
			}

			// log.Info("data:", _event.Data, "udata:", _event.Udata)
			// 可能的信息 syscall

			// flags:

			// EV_ADD                            = 0x1
			// EV_CLEAR                          = 0x20
			// EV_DELETE                         = 0x2
			// EV_DISABLE                        = 0x8
			// EV_DISPATCH                       = 0x80
			// EV_ENABLE                         = 0x4
			// EV_EOF                            = 0x8000
			// EV_ERROR                          = 0x4000
			// EV_FLAG0                          = 0x1000
			// EV_FLAG1                          = 0x2000
			// EV_ONESHOT                        = 0x10
			// EV_OOBAND                         = 0x2000
			// EV_POLL                           = 0x1000
			// EV_RECEIPT                        = 0x40
			// EV_SYSFLAGS                       = 0xf000

			// filters:

			// EVFILT_AIO                        = -0x3
			// EVFILT_FS                         = -0x9
			// EVFILT_MACHPORT                   = -0x8
			// EVFILT_PROC                       = -0x5
			// EVFILT_READ                       = -0x1
			// EVFILT_SIGNAL                     = -0x6
			// EVFILT_SYSCOUNT                   = 0xc
			// EVFILT_THREADMARKER               = 0xc
			// EVFILT_TIMER                      = -0x7
			// EVFILT_USER                       = -0xa
			// EVFILT_VM                         = -0xc
			// EVFILT_VNODE                      = -0x4
			// EVFILT_WRITE                      = -0x2

			var event common.Event

			if _event.Flags == (syscall.EV_ADD | syscall.EV_EOF) {
				// 连接关闭
				event = common.Event{
					Fd:   int32(_event.Ident),
					Type: common.EventClose,
				}
			} else {
				event = common.Event{
					Fd:   int32(_event.Ident),
					Type: common.EventIn,
				}
			}

			// 将事件发送给消费者处理
			eventGateway(event)
		}
	}

	return nil

}

func (p *Poll) RemoveAndClose(fd int) error {
	log.Info("===> 关闭连接")

	// 移除文件描述符的监听
	delEvent(fd, &p.changes)

	// 关闭文件描述符
	err := syscall.Close(fd)
	if err != nil {
		return err
	}

	return nil
}

func (p *Poll) Accept() (int, syscall.Sockaddr, error) {
	//ev_fd := int(event.Ident)
	nfd, sa, err := syscall.Accept(int(p.fd))
	log.Info("=====>", nfd, sa, err)
	if err != nil {
		return 0, nil, nil

	}
	return nfd, sa, nil

}

func (p *Poll) ShowChanges() string {
	return fmt.Sprint("\nchanges:", p.changes)
}

func delEvent(fd int, changes *[]syscall.Kevent_t) {
	length := len(*changes)
	if length == 1 && int((*changes)[0].Ident) == fd {
		*changes = (*changes)[:]
		return
	}
	for i := 0; i < length; i++ {
		if int((*changes)[i].Ident) == fd {
			if i == length-1 {
				*changes = (*changes)[0:i]
			} else if i == 0 {
				*changes = (*changes)[i+1:]
			} else {
				*changes = append((*changes)[0:i], (*changes)[i+1:]...)
			}
			return
		}
	}
}
