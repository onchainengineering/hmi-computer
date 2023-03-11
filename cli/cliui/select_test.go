package cliui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/cli/clibase"
	"github.com/coder/coder/cli/cliui"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/pty/ptytest"
)

func TestSelect(t *testing.T) {
	t.Parallel()
	t.Run("Select", func(t *testing.T) {
		t.Parallel()
		ptty := ptytest.New(t)
		msgChan := make(chan string)
		go func() {
			resp, err := newSelect(ptty, cliui.SelectOptions{
				Options: []string{"First", "Second"},
			})
			assert.NoError(t, err)
			msgChan <- resp
		}()
		require.Equal(t, "First", <-msgChan)
	})
}

func newSelect(ptty *ptytest.PTY, opts cliui.SelectOptions) (string, error) {
	value := ""
	cmd := &clibase.Cmd{
		Handler: func(inv *clibase.Invocation) error {
			var err error
			value, err = cliui.Select(inv, opts)
			return err
		},
	}
	inv := cmd.Invoke()
	ptty.Attach(inv)
	return value, inv.Run()
}

func TestRichSelect(t *testing.T) {
	t.Parallel()
	t.Run("RichSelect", func(t *testing.T) {
		t.Parallel()
		ptty := ptytest.New(t)
		msgChan := make(chan string)
		go func() {
			resp, err := newRichSelect(ptty, cliui.RichSelectOptions{
				Options: []codersdk.TemplateVersionParameterOption{
					{
						Name:        "A-Name",
						Value:       "A-Value",
						Description: "A-Description",
					}, {
						Name:        "B-Name",
						Value:       "B-Value",
						Description: "B-Description",
					},
				},
			})
			assert.NoError(t, err)
			msgChan <- resp
		}()
		require.Equal(t, "A-Value", <-msgChan)
	})
}

func newRichSelect(ptty *ptytest.PTY, opts cliui.RichSelectOptions) (string, error) {
	value := ""
	cmd := &clibase.Cmd{
		Handler: func(inv *clibase.Invocation) error {
			richOption, err := cliui.RichSelect(inv, opts)
			if err == nil {
				value = richOption.Value
			}
			return err
		},
	}
	inv := cmd.Invoke()
	ptty.Attach(inv)
	return value, inv.Run()
}
