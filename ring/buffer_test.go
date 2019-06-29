package ring_test

import (
	"errors"
	"io"
	"math/rand"
	"testing"

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

func TestBufferWriteReadCycles(t *testing.T) {
	// Get a ring buffer of capacity in 1, ..., 1024
	capacity := 1 + rand.Uint32()%1024
	buffer, err := ring.NewBuffer(capacity)
	require.Nil(t, err)

	for i := 0; i < 1024; i++ {
		// Generate a random byte slice to store in the buffer, of length in 0, ..., capacity, including full capacity.
		size := rand.Uint32() % (capacity + 1)
		value := make([]byte, size)
		n, err := rand.Read(value)
		assert.EqualValues(t, size, n)
		assert.Nil(t, err)

		// Nothing available for reading.
		assert.EqualValues(t, 0, buffer.Readable())

		// Write and read back.
		n, err = buffer.Write(value)
		require.Nil(t, err)
		require.EqualValues(t, size, n)
		assert.EqualValues(t, size, buffer.Readable())
		back := make([]byte, size)
		n, err = buffer.Read(back)
		require.Nil(t, err)
		assert.EqualValues(t, size, n)
		assert.Equal(t, value, back)
	}
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
