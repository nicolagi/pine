package ring

import (
	"errors"
)

var (
	ErrIllegal  = errors.New("illegal")
	ErrOverflow = errors.New("overflow")
)

type Buffer struct {
	contents     []byte
	readPointer  uint32
	writePointer uint32
	readable     uint32
	capacity     uint32
}

func NewBuffer() (*Buffer, error) {
	return &Buffer{
		capacity: 1,
		contents: make([]byte, 4),
	}, nil
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	for ; n < len(p); n++ {
		if b.readable == 0 {
			err = ErrOverflow
			break
		}
		p[n] = b.contents[b.readPointer]
		b.readPointer = (b.readPointer + 1) % b.capacity
		b.readable--
	}
	return
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	for ; n < len(p); n++ {
		if b.readable == b.capacity {
			b.realloc()
		}
		b.contents[b.writePointer] = p[n]
		b.writePointer = (b.writePointer + 1) % b.capacity
		b.readable++
	}
	return
}

func (b *Buffer) Readable() uint32 {
	return b.readable
}

func (b *Buffer) realloc() {
	newb := &Buffer{
		capacity: 2 * b.capacity,
		contents: make([]byte, 2*b.capacity),
	}
	p := make([]byte, b.Readable())
	b.Read(p)
	newb.Write(p)
	*b = *newb
}
