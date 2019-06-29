package ring

import (
	"bytes"
	"testing"

	"github.com/lionkov/go9p/p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageBuffer(t *testing.T) {
	const msize uint32 = 8192
	// Input to the message buffer, expected output from the message buffer
	// for the given input.
	inb, outs := func() ([]byte, string) {
		fc := p.NewFcall(msize)
		require.Nil(t, p.PackTversion(fc, msize, "9P2000"))
		return fc.Pkt, fc.String() + "\n"
	}()
	t.Run("ingest and print entire message", func(t *testing.T) {
		mb, err := NewMessageBuffer(msize)
		require.Nil(t, err)
		mb.Ingest(inb)
		out := bytes.NewBuffer(nil)
		mb.PrintMessages(out)
		assert.Equal(t, outs, out.String())
	})
	t.Run("ingest message a bit at a time then print it", func(t *testing.T) {
		mb, err := NewMessageBuffer(msize)
		require.Nil(t, err)
		i := 0
		for ; i < len(inb)-4; i += 2 {
			mb.Ingest(inb[i : i+2])
			// Following would panic() if output write was used.
			mb.PrintMessages(nil)
		}
		mb.Ingest(inb[i:])
		out := bytes.NewBuffer(nil)
		mb.PrintMessages(out)
		assert.Equal(t, outs, out.String())
	})
	t.Run("ingest two messages in one go and print both", func(t *testing.T) {
		mb, err := NewMessageBuffer(msize)
		require.Nil(t, err)
		mb.Ingest(inb)
		mb.Ingest(inb)
		out := bytes.NewBuffer(nil)
		mb.PrintMessages(out)
		assert.Equal(t, outs+outs, out.String())
	})
}
