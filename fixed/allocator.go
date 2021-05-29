package fixed

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	// ErrOverflow returned if requested buffer number larget then max number.
	ErrOverflow = errors.New("buffer number overflow")
)

var iovecSize = int(unsafe.Sizeof(unix.Iovec{}))

type allocator struct {
	max        int // max number of buffers
	bufferSize int // requested size of the buffer

	// mem is splitted in two parts
	// header - list of iovec structs.
	// starts at mem[0]. current length is iovecSz*allocated
	// buffers - list of buffers of the same size.
	mem []byte

	reg BufferRegistry
}

func (a *allocator) init() error {
	prot := unix.PROT_READ | unix.PROT_WRITE
	flags := unix.MAP_ANON | unix.MAP_PRIVATE
	size := a.bufferSize * a.max
	mem, err := unix.Mmap(-1, 0, size, prot, flags)
	if err != nil {
		return err
	}
	a.mem = mem
	iovec := []unix.Iovec{{Base: &mem[0], Len: uint64(size)}}
	return a.reg.RegisterBuffers(iovec)
}

func (a *allocator) close() error {
	return unix.Munmap(a.mem)
}

func (a *allocator) bufAt(pos int) []byte {
	start := pos * a.bufferSize
	return a.mem[start : start+a.bufferSize]
}
