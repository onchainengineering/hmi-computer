package codersdk

import (
	"context"
	"encoding/json"
	"net/http"
)

var (
	// ExperimentVSCodeLocal enables a workspace button to launch VSCode
	// and connect using the local VSCode extension.
	ExperimentVSCodeLocal = "vscode_local"
	// ExperimentsAll should include all experiments that are safe for
	// users to opt-in to via --experimental='*'.
	// Experiments that are not ready for consumption by all users should
	// not be included here.
	ExperimentsAll Experiments = Experiments{
		ExperimentVSCodeLocal,
	}
)

// Experiments is a list of experiments that are enabled for the deployment.
// Multiple experiments may be enabled at the same time.
// Experiments are not safe for production use, and are not guaranteed to
// be backwards compatible. They may be removed or renamed at any time.
type Experiments []string

func (e Experiments) Enabled(ex string) bool {
	for _, v := range e {
		if v == ex {
			return true
		}
	}
	return false
}

func (c *Client) Experiments(ctx context.Context) (Experiments, error) {
	res, err := c.Request(ctx, http.MethodGet, "/api/v2/experiments", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, readBodyAsError(res)
	}
	var exp []string
	return exp, json.NewDecoder(res.Body).Decode(&exp)
}
