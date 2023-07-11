package codersdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

const (
	LicenseExpiryClaim = "license_expires"
)

type AddLicenseRequest struct {
	License string `json:"license" validate:"required"`
}

type License struct {
	ID         int32     `json:"id" table:"id,default_sort"`
	UUID       uuid.UUID `json:"uuid" table:"uuid" format:"uuid"`
	UploadedAt time.Time `json:"uploaded_at" table:"uploaded_at" format:"date-time"`
	// Claims are the JWT claims asserted by the license.  Here we use
	// a generic string map to ensure that all data from the server is
	// parsed verbatim, not just the fields this version of Coder
	// understands.
	Claims map[string]interface{} `json:"claims" table:"claims"`
}

// ExpiresAt returns the expiration time of the license.
// If the claim is missing or has an unexpected type, an error is returned.
func (l *License) ExpiresAt() (time.Time, error) {
	expClaim, ok := l.Claims[LicenseExpiryClaim]
	if !ok {
		return time.Time{}, xerrors.New("license_expires claim is missing")
	}

	// The claim could be a unix timestamp or a RFC3339 formatted string.
	// Everything is already an interface{}, so we need to do some type
	// assertions to figure out what we're dealing with.
	if unix, ok := expClaim.(json.Number); ok {
		i64, err := unix.Int64()
		if err != nil {
			return time.Time{}, xerrors.Errorf("license_expires claim is not a valid unix timestamp: %w", err)
		}
		return time.Unix(i64, 0), nil
	}

	if str, ok := expClaim.(string); ok {
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return time.Time{}, xerrors.Errorf("license_expires claim is not a valid RFC3339 timestamp: %w", err)
		}
		return t, nil
	}

	return time.Time{}, xerrors.Errorf("license_expires claim has unexpected type %T", expClaim)
}

// Features provides the feature claims in license.
func (l *License) Features() (map[FeatureName]int64, error) {
	strMap, ok := l.Claims["features"].(map[string]interface{})
	if !ok {
		return nil, xerrors.New("features key is unexpected type")
	}
	fMap := make(map[FeatureName]int64)
	for k, v := range strMap {
		jn, ok := v.(json.Number)
		if !ok {
			return nil, xerrors.Errorf("feature %q has unexpected type", k)
		}

		n, err := jn.Int64()
		if err != nil {
			return nil, err
		}

		fMap[FeatureName(k)] = n
	}

	return fMap, nil
}

func (c *Client) AddLicense(ctx context.Context, r AddLicenseRequest) (License, error) {
	res, err := c.Request(ctx, http.MethodPost, "/api/v2/licenses", r)
	if err != nil {
		return License{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return License{}, ReadBodyAsError(res)
	}
	var l License
	d := json.NewDecoder(res.Body)
	d.UseNumber()
	return l, d.Decode(&l)
}

func (c *Client) Licenses(ctx context.Context) ([]License, error) {
	res, err := c.Request(ctx, http.MethodGet, "/api/v2/licenses", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, ReadBodyAsError(res)
	}
	var licenses []License
	d := json.NewDecoder(res.Body)
	d.UseNumber()
	return licenses, d.Decode(&licenses)
}

func (c *Client) DeleteLicense(ctx context.Context, id int32) error {
	res, err := c.Request(ctx, http.MethodDelete, fmt.Sprintf("/api/v2/licenses/%d", id), nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return ReadBodyAsError(res)
	}
	return nil
}
