package ring

import (
	"errors"
	"fmt"
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

func NewBuffer(capacity uint32) (*Buffer, error) {
	if capacity == 0 {
		return nil, fmt.Errorf("zero capacity: %w", ErrIllegal)
	}
	return &Buffer{
		capacity: capacity,
		contents: make([]byte, capacity),
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
			err = ErrOverflow
			break
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
