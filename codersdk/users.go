package codersdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coder/coder/coderd"
)

func (c *Client) CreateInitialUser(ctx context.Context, req coderd.CreateUserRequest) (coderd.User, error) {
	res, err := c.request(ctx, http.MethodPost, "/api/v2/user", req)
	if err != nil {
		return coderd.User{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return coderd.User{}, readBodyAsError(res)
	}
	var user coderd.User
	return user, json.NewDecoder(res.Body).Decode(&user)
}

// User returns a user for the ID provided.
// If the ID string is empty, the current user will be returned.
func (c *Client) User(ctx context.Context, id string) (coderd.User, error) {
	res, err := c.request(ctx, http.MethodGet, "/api/v2/user", nil)
	if err != nil {
		return coderd.User{}, err
	}
	defer res.Body.Close()
	if res.StatusCode > http.StatusOK {
		return coderd.User{}, readBodyAsError(res)
	}
	var user coderd.User
	return user, json.NewDecoder(res.Body).Decode(&user)
}

func (c *Client) LoginWithPassword(ctx context.Context, req coderd.LoginWithPasswordRequest) error {
	res, err := c.request(ctx, http.MethodPost, "/api/v2/login", req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		fmt.Printf("Are we reading here?\n")
		return readBodyAsError(res)
	}
	var resp coderd.LoginWithPasswordResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return err
	}
	return c.setSessionToken(resp.SessionToken)
}
