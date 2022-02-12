package expect_test

import (
	"errors"
	"io"

	//"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	. "github.com/coder/coder/expect"
)

func TestPassthroughPipe(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()

	passthroughPipe, err := NewPassthroughPipe(pipeReader)
	require.NoError(t, err)

	err = passthroughPipe.SetReadDeadline(time.Now().Add(time.Hour))
	require.NoError(t, err)

	pipeError := errors.New("pipe error")
	err = pipeWriter.CloseWithError(pipeError)
	require.NoError(t, err)

	p := make([]byte, 1)
	_, err = passthroughPipe.Read(p)
	require.Equal(t, err, pipeError)
}

// TODO(Bryan): Can this be enabled on Windows?
// func TestPassthroughPipeTimeout(t *testing.T) {
// 	r, w := io.Pipe()

// 	passthroughPipe, err := NewPassthroughPipe(r)
// 	require.NoError(t, err)

// 	err = passthroughPipe.SetReadDeadline(time.Now())
// 	require.NoError(t, err)

// 	_, err = w.Write([]byte("a"))
// 	require.NoError(t, err)

// 	p := make([]byte, 1)
// 	_, err = passthroughPipe.Read(p)
// 	require.True(t, os.IsTimeout(err))

// 	err = passthroughPipe.SetReadDeadline(time.Time{})
// 	require.NoError(t, err)

// 	n, err := passthroughPipe.Read(p)
// 	require.Equal(t, 1, n)
// 	require.NoError(t, err)
// }
