package uring

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	MinSize = 2
	MaxSize = 4096
)

func Setup(size uint, params *IOUringParams) (*Ring, error) {
	var ring Ring
	if params != nil {
		ring.params = *params
	}
	if err := setup(&ring, size, &ring.params); err != nil {
		return nil, err
	}
	return &ring, nil
}

func setup(ring *Ring, size uint, p *IOUringParams) error {
	fd, _, errno := unix.Syscall(IO_URING_SETUP, uintptr(size), uintptr(unsafe.Pointer(p)), 0)
	if errno != 0 {
		return fmt.Errorf("IO_URING_SETUP %w", error(errno))
	}
	ring.fd = int(fd)

	sqsize := p.SQOff.Array + p.SQEntries*uint32(4)
	cqsize := p.CQOff.CQEs + p.CQEntries*uint32(cqeSize)
	isSingleMap := p.Features&IORING_FEAT_SINGLE_MMAP > 0
	if isSingleMap {
		if cqsize > sqsize {
			sqsize = cqsize
		}
	}

	data, err := unix.Mmap(int(fd), IORING_OFF_SQ_RING, int(sqsize),
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		return err
	}
	ring.sqData = data
	pointer := unsafe.Pointer(&data[0])

	ring.sq.head = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.SQOff.Head)))
	ring.sq.tail = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.SQOff.Tail)))
	ring.sq.ringMask = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.SQOff.RingMask)))
	ring.sq.ringEntries = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.SQOff.RingEntries)))
	ring.sq.flags = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.SQOff.Flags)))
	ring.sq.dropped = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.SQOff.Dropped)))
	ring.sq.array = uint32Array(unsafe.Pointer(uintptr(pointer) + uintptr(p.SQOff.Array)))

	if !isSingleMap {
		data, err = unix.Mmap(int(fd), IORING_OFF_CQ_RING, int(cqsize),
			unix.PROT_READ|unix.PROT_WRITE,
			unix.MAP_SHARED|unix.MAP_POPULATE)
		if err != nil {
			return err
		}
		ring.cqData = data
		pointer = unsafe.Pointer(&data[0])
	}

	ring.cq.head = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.CQOff.Head)))
	ring.cq.tail = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.CQOff.Tail)))
	ring.cq.ringmask = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.CQOff.RingMask)))
	ring.cq.ringEntries = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.CQOff.RingEntries)))
	ring.cq.overflow = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.CQOff.Overflow)))
	ring.cq.cqes = cqeArray(unsafe.Pointer(uintptr(pointer) + uintptr(p.CQOff.CQEs)))
	if p.CQOff.Flags != 0 {
		ring.cq.flags = (*uint32)(unsafe.Pointer(uintptr(pointer) + uintptr(p.CQOff.Flags)))
	}

	entries, err := unix.Mmap(int(fd), IORING_OFF_SQES,
		int(p.SQEntries)*int(sqeSize),
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_SHARED|unix.MAP_POPULATE)

	if err != nil {
		return err
	}
	ring.sqArrayData = entries
	ring.sq.sqes = sqeArray(unsafe.Pointer(&entries[0]))
	return nil
}

func (r *Ring) Close() (err error) {
	if r.cqData != nil {
		ret := unix.Munmap(r.cqData)
		if err == nil {
			err = ret
		}
		if ret == nil {
			r.cqData = nil
		}
	}
	if r.sqData != nil {
		ret := unix.Munmap(r.sqData)
		if err == nil {
			err = ret
		}
		if ret == nil {
			r.sqData = nil
		}
	}
	if r.sqArrayData != nil {
		ret := unix.Munmap(r.sqArrayData)
		if err == nil {
			err = ret
		}
		if ret == nil {
			r.sqArrayData = nil
		}
	}
	if r.fd != 0 {
		ret := unix.Close(r.fd)
		if err == nil {
			err = ret
		}
		if ret == nil {
			r.fd = 0
		}
	}
	return
}
