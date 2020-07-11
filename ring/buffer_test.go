package ring_test

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/nicolagi/pine/ring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufferImplementsReaderWriter(t *testing.T) {
	var _ io.Reader = (*ring.Buffer)(nil)
	var _ io.Writer = (*ring.Buffer)(nil)
}

func TestBufferReadOverflow(t *testing.T) {
	buffer, err := ring.NewBuffer()
	require.Nil(t, err)
	n, err := buffer.Write([]byte("hello"))
	assert.Nil(t, err)
	assert.Equal(t, 5, n)
	back := make([]byte, 8)
	n, err = buffer.Read(back)
	assert.Equal(t, 5, n)
	assert.Equal(t, append([]byte("hello"), 0, 0, 0), back)
	assert.True(t, errors.Is(err, ring.ErrOverflow))
}

type opType uint8

const (
	opRead  opType = 0
	opWrite opType = 1
)

type op struct {
	t opType // opRead or opWrite
	n int    // for read: how many bytes to read
	p []byte // for write: the bytes to write
}

func (op *op) String() string {
	switch op.t {
	case opRead:
		return fmt.Sprintf("read of %d bytes", op.n)
	case opWrite:
		return fmt.Sprintf("write of %d bytes, %.8x", len(op.p), op.p)
	default:
		panic("unknown op type")
	}
}

type opOutput struct {
	n   int    // for read and write: the number of bytes read/written
	err error  // for read and write
	p   []byte // only for read: the bytes that were read
}

func (op *op) applyToRingBuffer(rb *ring.Buffer) *opOutput {
	switch op.t {
	case opRead:
		p := make([]byte, op.n)
		n, err := rb.Read(p)
		return &opOutput{n: n, err: err, p: p}
	case opWrite:
		n, err := rb.Write(op.p)
		return &opOutput{n: n, err: err}
	default:
		panic("unknown op type")
	}
}

type file struct {
	osf         *os.File
	readOffset  int64
	writeOffset int64
}

func (op *op) applyToFile(f *file) *opOutput {
	switch op.t {
	case opRead:
		p := make([]byte, op.n)
		n, err := f.osf.ReadAt(p, f.readOffset)
		f.readOffset += int64(n)
		return &opOutput{n: n, err: err, p: p}
	case opWrite:
		n, err := f.osf.WriteAt(op.p, f.writeOffset)
		f.writeOffset += int64(n)
		return &opOutput{n: n, err: err}
	default:
		panic("unknown op type")
	}
}

func TestRingBufferWhatYouWriteIsWhatYouRead(t *testing.T) {
	const maxBufferSize = 4096
	dir := t.TempDir()
	availableForReading := 0
	buffer, err := ring.NewBuffer()
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "readwrite"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	refImpl := &file{osf: f}
	err = quick.CheckEqual(func(op *op) *opOutput {
		t.Log(op)
		return op.applyToRingBuffer(buffer)
	}, func(op *op) *opOutput {
		return op.applyToFile(refImpl)
	}, &quick.Config{
		Values: func(values []reflect.Value, rand *rand.Rand) {
			for i := 0; i < len(values); i++ {
				var nextOp op
				nextOp.t = opType(rand.Int() % 2)
				switch nextOp.t {
				case opRead:
					if got, want := buffer.Readable(), availableForReading; got != uint32(want) {
						t.Errorf("got %d, want %d readable bytes", got, want)
					}
					nextOp.n = rand.Intn(availableForReading + 1)
					availableForReading -= nextOp.n
				case opWrite:
					sz := rand.Intn(maxBufferSize - availableForReading + 1)
					availableForReading += sz
					nextOp.p = make([]byte, sz)
					rand.Read(nextOp.p)
				default:
					panic("unknown op type")
				}
				values[i] = reflect.ValueOf(&nextOp)
			}
		},
	})
	if err != nil {
		t.Error(err)
	}
}
