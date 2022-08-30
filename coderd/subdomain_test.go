package coderd_test

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd"
)

func TestParseSubdomainAppURL(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name          string
		URL           string
		Expected      coderd.ApplicationURL
		ExpectedError string
	}{
		{
			Name:          "Empty",
			URL:           "https://example.com",
			Expected:      coderd.ApplicationURL{},
			ExpectedError: "invalid application url format",
		},
		{
			Name:          "Workspace.Agent+App",
			URL:           "https://workspace.agent--app.coder.com",
			Expected:      coderd.ApplicationURL{},
			ExpectedError: "invalid application url format",
		},
		{
			Name:          "Workspace+App",
			URL:           "https://workspace--app.coder.com",
			Expected:      coderd.ApplicationURL{},
			ExpectedError: "invalid application url format",
		},
		// Correct
		{
			Name: "User+Workspace+App",
			URL:  "https://user--workspace--app.coder.com",
			Expected: coderd.ApplicationURL{
				AppName:       "app",
				WorkspaceName: "workspace",
				Agent:         "",
				Username:      "user",
				Path:          "",
				Domain:        "coder.com",
			},
		},
		{
			Name: "User+Workspace+Port",
			URL:  "https://user--workspace--8080.coder.com",
			Expected: coderd.ApplicationURL{
				AppName:       "8080",
				WorkspaceName: "workspace",
				Agent:         "",
				Username:      "user",
				Path:          "",
				Domain:        "coder.com",
			},
		},
		{
			Name: "User+Workspace.Agent+App",
			URL:  "https://user--workspace--agent--app.coder.com",
			Expected: coderd.ApplicationURL{
				AppName:       "app",
				WorkspaceName: "workspace",
				Agent:         "agent",
				Username:      "user",
				Path:          "",
				Domain:        "coder.com",
			},
		},
		{
			Name: "User+Workspace.Agent+Port",
			URL:  "https://user--workspace--agent--8080.coder.com",
			Expected: coderd.ApplicationURL{
				AppName:       "8080",
				WorkspaceName: "workspace",
				Agent:         "agent",
				Username:      "user",
				Path:          "",
				Domain:        "coder.com",
			},
		},
		{
			Name: "HyphenatedNames",
			URL:  "https://admin-user--workspace-thing--agent-thing--app-name.coder.com",
			Expected: coderd.ApplicationURL{
				AppName:       "app-name",
				WorkspaceName: "workspace-thing",
				Agent:         "agent-thing",
				Username:      "admin-user",
				Path:          "",
				Domain:        "coder.com",
			},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("GET", c.URL, nil)

			app, err := coderd.ParseSubdomainAppURL(r)
			if c.ExpectedError == "" {
				require.NoError(t, err)
				require.Equal(t, c.Expected, app, "expected app")
			} else {
				require.ErrorContains(t, err, c.ExpectedError, "expected error")
			}
		})
	}
}
