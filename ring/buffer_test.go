package ring_test

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
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

func TestBufferIllegalCapacity(t *testing.T) {
	buffer, err := ring.NewBuffer(0)
	require.Nil(t, buffer)
	require.True(t, errors.Is(err, ring.ErrIllegal))
}

func TestBufferWriteOverflow(t *testing.T) {
	buffer, err := ring.NewBuffer(4)
	require.Nil(t, err)
	n, err := buffer.Write([]byte("hello"))
	assert.Equal(t, 4, n)
	assert.True(t, errors.Is(err, ring.ErrOverflow))
	back := make([]byte, 4)
	n, err = buffer.Read(back)
	assert.Nil(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, []byte("hell"), back)
}

func TestBufferReadOverflow(t *testing.T) {
	buffer, err := ring.NewBuffer(8)
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
	bufferSizes := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384}
	for _, bufferSize := range bufferSizes {
		t.Run(fmt.Sprintf("buffer-size-%d", bufferSize), func(t *testing.T) {
			availableForReading := 0
			buffer, err := ring.NewBuffer(uint32(bufferSize))
			if err != nil {
				t.Fatal(err)
			}
			f, err := ioutil.TempFile("", "pine-ring-")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = f.Close()
				_ = os.Remove(f.Name())
			}()
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
							sz := rand.Intn(bufferSize - availableForReading + 1)
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
		})
	}
}
