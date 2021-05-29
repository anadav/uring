package loop

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func newPoll(n int) (*poll, error) {
	p, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &poll{fd: p, events: make([]unix.EpollEvent, n)}, nil
}

type poll struct {
	fd int // epoll fd

	buf    [8]byte
	events []unix.EpollEvent
}

func (p *poll) addRead(fd int32) error {
	return unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, int(fd),
		&unix.EpollEvent{
			Fd:     fd,
			Events: unix.EPOLLIN,
		},
	)
}

func (p *poll) wait(iter func(int32)) error {
	for {
		n, err := unix.EpollWait(p.fd, p.events, -1)
		if err == unix.EINTR {
			continue
		}
		for i := 0; i < n; i++ {
			_, err := unix.Read(int(p.events[i].Fd), p.buf[:])
			if err != nil {
				panic(err)
			}
			// uint64 in the machine native order
			cnt := *(*uint64)(unsafe.Pointer(&p.buf))
			for j := uint64(0); j < cnt; j++ {
				iter(p.events[i].Fd)
			}
		}
		return err
	}
}

func (p *poll) close() error {
	return unix.Close(p.fd)
}
