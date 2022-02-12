package expect_test

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/coder/coder/expect"
)

func TestReaderLease(t *testing.T) {
	in, out := io.Pipe()
	defer out.Close()
	defer in.Close()

	readerLease := NewReaderLease(in)

	tests := []struct {
		title    string
		expected string
	}{
		{
			"Read cancels with deadline",
			"apple",
		},
		{
			"Second read has no bytes stolen",
			"banana",
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			tin, tout := io.Pipe()

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				io.Copy(tout, readerLease.NewReader(ctx))
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := out.Write([]byte(test.expected))
				require.Nil(t, err)
			}()

			for i := 0; i < len(test.expected); i++ {
				p := make([]byte, 1)
				n, err := tin.Read(p)
				require.Nil(t, err)
				require.Equal(t, 1, n)
				require.Equal(t, test.expected[i], p[0])
			}

			cancel()
			wg.Wait()
		})
	}
}
