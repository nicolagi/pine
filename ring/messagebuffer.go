package ring

import (
	"fmt"
	"io"

	"github.com/lionkov/go9p/p"
	log "github.com/sirupsen/logrus"
)

const (
	unknownSize uint32 = 0xffffffff
)

type MessageBuffer struct {
	buffer     *Buffer
	curSize    uint32
	curMessage []byte
}

func NewMessageBuffer() (*MessageBuffer, error) {
	buffer, err := NewBuffer()
	if err != nil {
		return nil, err
	}
	return &MessageBuffer{
		buffer:     buffer,
		curSize:    unknownSize,
		curMessage: make([]byte, 4),
	}, nil
}

func (mb *MessageBuffer) Ingest(chunk []byte) error {
	n, err := mb.buffer.Write(chunk)
	if err != nil {
		return fmt.Errorf("ingest ring buffer write: %w", err)
	}
	if n != len(chunk) {
		return fmt.Errorf("could only write %d of %d bytes", n, len(chunk))
	}
	return nil
}

func (mb *MessageBuffer) PrintMessages(out io.Writer) error {
next:
	// If we don't know the size of the current message and we have at least 4 bytes,
	// decode the size of the current message.
	if mb.curSize == unknownSize && mb.buffer.Readable() >= 4 {
		sizeBytes := mb.curMessage[:4]
		n, err := mb.buffer.Read(sizeBytes)
		if err != nil {
			return fmt.Errorf("ring buffer read: %w", err)
		}
		if n != 4 {
			return fmt.Errorf("could only read %d of 4 bytes", n)
		}
		mb.curSize, _ = p.Gint32(sizeBytes)
		if uint32(len(mb.curMessage)) < mb.curSize {
			p := make([]byte, mb.curSize)
			copy(p, mb.curMessage)
			mb.curMessage = p
		}
	}
	// If we have all the current message, decode that.
	if mb.buffer.Readable()+4 >= mb.curSize {
		n, err := mb.buffer.Read(mb.curMessage[4:mb.curSize])
		if err != nil {
			return fmt.Errorf("ring buffer read: %w", err)
		}
		if n < 0 || uint32(n) != mb.curSize-4 {
			return fmt.Errorf("could only read %d of %d bytes", n, mb.curSize-4)
		}
		fc, err, _ := p.Unpack(mb.curMessage[:mb.curSize], false)
		if err != nil {
			log.WithField("cause", err).Error("Could not unpack message")
		} else {
			fmt.Fprintln(out, fc)
		}
		mb.curSize = unknownSize
		// Go back to the beginning, we might have room for another message,
		// or perhaps just its size.
		goto next
	}
	return nil
}
