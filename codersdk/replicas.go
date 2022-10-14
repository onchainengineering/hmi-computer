package codersdk

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

type Replica struct {
	// ID is the unique identifier for the replica.
	ID uuid.UUID `json:"id"`
	// Hostname is the hostname of the replica.
	Hostname string `json:"hostname"`
	// CreatedAt is when the replica was first seen.
	CreatedAt time.Time `json:"created_at"`
	// RelayAddress is the accessible address to relay DERP connections.
	RelayAddress string `json:"relay_address"`
	// RegionID is the region of the replica.
	RegionID int32 `json:"region_id"`
	// Error is the error.
	Error string `json:"error"`
}

// Replicas fetches the list of replicas.
func (c *Client) Replicas(ctx context.Context) ([]Replica, error) {
	res, err := c.Request(ctx, http.MethodGet, "/api/v2/replicas", nil)
	if err != nil {
		return nil, xerrors.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, readBodyAsError(res)
	}

	var replicas []Replica
	return replicas, json.NewDecoder(res.Body).Decode(&replicas)
}
