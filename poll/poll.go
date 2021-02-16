package poll

import "github.com/cofepy/gn/common"

type PollInterface interface {
	RemoveAndClose(fd int) error
	Wait(handler func(event common.Event)) error
	AddRead(fd int) error
}
