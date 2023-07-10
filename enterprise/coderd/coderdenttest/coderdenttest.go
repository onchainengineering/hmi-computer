package coderdenttest

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/enterprise/coderd"
	"github.com/coder/coder/enterprise/coderd/license"
)

const (
	testKeyID = "enterprise-test"
)

var (
	testPrivateKey ed25519.PrivateKey
	testPublicKey  ed25519.PublicKey

	Keys = map[string]ed25519.PublicKey{}
)

func init() {
	var err error
	testPublicKey, testPrivateKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	Keys[testKeyID] = testPublicKey
}

type Options struct {
	*coderdtest.Options
	AuditLogging                bool
	BrowserOnly                 bool
	EntitlementsUpdateInterval  time.Duration
	SCIMAPIKey                  []byte
	UserWorkspaceQuota          int
	ProxyHealthInterval         time.Duration
	NoDefaultQuietHoursSchedule bool
}

// New constructs a codersdk client connected to an in-memory Enterprise API instance.
func New(t *testing.T, options *Options) *codersdk.Client {
	client, _, _ := NewWithAPI(t, options)
	return client
}

func NewWithAPI(t *testing.T, options *Options) (*codersdk.Client, io.Closer, *coderd.API) {
	t.Helper()

	if options == nil {
		options = &Options{}
	}
	if options.Options == nil {
		options.Options = &coderdtest.Options{}
	}
	setHandler, cancelFunc, serverURL, oop := coderdtest.NewOptions(t, options.Options)
	if !options.NoDefaultQuietHoursSchedule && oop.DeploymentValues.UserQuietHoursSchedule.DefaultSchedule.Value() == "" {
		err := oop.DeploymentValues.UserQuietHoursSchedule.DefaultSchedule.Set("0 0 * * *")
		require.NoError(t, err)
	}
	coderAPI, err := coderd.New(context.Background(), &coderd.Options{
		RBAC:                       true,
		AuditLogging:               options.AuditLogging,
		BrowserOnly:                options.BrowserOnly,
		SCIMAPIKey:                 options.SCIMAPIKey,
		DERPServerRelayAddress:     oop.AccessURL.String(),
		DERPServerRegionID:         oop.DERPMap.RegionIDs()[0],
		Options:                    oop,
		EntitlementsUpdateInterval: options.EntitlementsUpdateInterval,
		Keys:                       Keys,
		ProxyHealthInterval:        options.ProxyHealthInterval,
		DefaultQuietHoursSchedule:  oop.DeploymentValues.UserQuietHoursSchedule.DefaultSchedule.Value(),
		QuietHoursWindowDuration:   oop.DeploymentValues.UserQuietHoursSchedule.WindowDuration.Value(),
	})
	require.NoError(t, err)
	setHandler(coderAPI.AGPL.RootHandler)
	var provisionerCloser io.Closer = nopcloser{}
	if options.IncludeProvisionerDaemon {
		provisionerCloser = coderdtest.NewProvisionerDaemon(t, coderAPI.AGPL)
	}

	t.Cleanup(func() {
		cancelFunc()
		_ = provisionerCloser.Close()
		_ = coderAPI.Close()
	})
	client := codersdk.New(serverURL)
	client.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				//nolint:gosec
				InsecureSkipVerify: true,
			},
		},
	}
	return client, provisionerCloser, coderAPI
}

type LicenseOptions struct {
	AccountType string
	AccountID   string
	Trial       bool
	AllFeatures bool
	GraceAt     time.Time
	ExpiresAt   time.Time
	Features    license.Features
}

// AddFullLicense generates a license with all features enabled.
func AddFullLicense(t *testing.T, client *codersdk.Client) codersdk.License {
	return AddLicense(t, client, LicenseOptions{AllFeatures: true})
}

// AddLicense generates a new license with the options provided and inserts it.
func AddLicense(t *testing.T, client *codersdk.Client, options LicenseOptions) codersdk.License {
	l, err := client.AddLicense(context.Background(), codersdk.AddLicenseRequest{
		License: GenerateLicense(t, options),
	})
	require.NoError(t, err)
	return l
}

// GenerateLicense returns a signed JWT using the test key.
func GenerateLicense(t *testing.T, options LicenseOptions) string {
	if options.ExpiresAt.IsZero() {
		options.ExpiresAt = time.Now().Add(time.Hour)
	}
	if options.GraceAt.IsZero() {
		options.GraceAt = time.Now().Add(time.Hour)
	}

	c := &license.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    "test@testing.test",
			ExpiresAt: jwt.NewNumericDate(options.ExpiresAt),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
		LicenseExpires: jwt.NewNumericDate(options.GraceAt),
		AccountType:    options.AccountType,
		AccountID:      options.AccountID,
		Trial:          options.Trial,
		Version:        license.CurrentVersion,
		AllFeatures:    options.AllFeatures,
		Features:       options.Features,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, c)
	tok.Header[license.HeaderKeyID] = testKeyID
	signedTok, err := tok.SignedString(testPrivateKey)
	require.NoError(t, err)
	return signedTok
}

type nopcloser struct{}

func (nopcloser) Close() error { return nil }
