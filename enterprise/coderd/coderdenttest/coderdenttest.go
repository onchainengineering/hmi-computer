package coderdenttest

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/onchainengineering/hmi-computer/v2/coderd/coderdtest"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/coderd/database"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/coderd/database/dbmem"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/coderd/database/pubsub"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/codersdk"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/codersdk/drpc"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/enterprise/coderd"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/enterprise/coderd/license"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/enterprise/dbcrypt"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/provisioner/echo"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/provisionerd"
	provisionerdproto "github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/provisionerd/proto"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/provisionersdk"
	sdkproto "github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/provisionersdk/proto"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/testutil"
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
	AuditLogging               bool
	BrowserOnly                bool
	EntitlementsUpdateInterval time.Duration
	SCIMAPIKey                 []byte
	UserWorkspaceQuota         int
	ProxyHealthInterval        time.Duration
	LicenseOptions             *LicenseOptions
	DontAddLicense             bool
	DontAddFirstUser           bool
	ReplicaSyncUpdateInterval  time.Duration
	ReplicaErrorGracePeriod    time.Duration
	ExternalTokenEncryption    []dbcrypt.Cipher
	ProvisionerDaemonPSK       string
}

// New constructs a codersdk client connected to an in-memory Enterprise API instance.
func New(t *testing.T, options *Options) (*codersdk.Client, codersdk.CreateFirstUserResponse) {
	client, _, _, user := NewWithAPI(t, options)
	return client, user
}

func NewWithDatabase(t *testing.T, options *Options) (*codersdk.Client, database.Store, codersdk.CreateFirstUserResponse) {
	client, _, api, user := NewWithAPI(t, options)
	return client, api.Database, user
}

func NewWithAPI(t *testing.T, options *Options) (
	*codersdk.Client, io.Closer, *coderd.API, codersdk.CreateFirstUserResponse,
) {
	t.Helper()

	if options == nil {
		options = &Options{}
	}
	if options.Options == nil {
		options.Options = &coderdtest.Options{}
	}
	require.False(t, options.DontAddFirstUser && !options.DontAddLicense, "DontAddFirstUser requires DontAddLicense")
	setHandler, cancelFunc, serverURL, oop := coderdtest.NewOptions(t, options.Options)
	coderAPI, err := coderd.New(context.Background(), &coderd.Options{
		RBAC:                       true,
		AuditLogging:               options.AuditLogging,
		BrowserOnly:                options.BrowserOnly,
		SCIMAPIKey:                 options.SCIMAPIKey,
		DERPServerRelayAddress:     oop.AccessURL.String(),
		DERPServerRegionID:         oop.BaseDERPMap.RegionIDs()[0],
		ReplicaSyncUpdateInterval:  options.ReplicaSyncUpdateInterval,
		ReplicaErrorGracePeriod:    options.ReplicaErrorGracePeriod,
		Options:                    oop,
		EntitlementsUpdateInterval: options.EntitlementsUpdateInterval,
		LicenseKeys:                Keys,
		ProxyHealthInterval:        options.ProxyHealthInterval,
		DefaultQuietHoursSchedule:  oop.DeploymentValues.UserQuietHoursSchedule.DefaultSchedule.Value(),
		ProvisionerDaemonPSK:       options.ProvisionerDaemonPSK,
		ExternalTokenEncryption:    options.ExternalTokenEncryption,
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
	var user codersdk.CreateFirstUserResponse
	if !options.DontAddFirstUser {
		user = coderdtest.CreateFirstUser(t, client)
		if !options.DontAddLicense {
			lo := LicenseOptions{}
			if options.LicenseOptions != nil {
				lo = *options.LicenseOptions
				// The pgCoord is not supported by the fake DB & in-memory Pubsub.  It only works on a real postgres.
				if lo.AllFeatures || (lo.Features != nil && lo.Features[codersdk.FeatureHighAvailability] != 0) {
					// we check for the in-memory test types so that the real types don't have to exported
					_, ok := coderAPI.Pubsub.(*pubsub.MemoryPubsub)
					require.False(t, ok, "FeatureHighAvailability is incompatible with MemoryPubsub")
					_, ok = coderAPI.Database.(*dbmem.FakeQuerier)
					require.False(t, ok, "FeatureHighAvailability is incompatible with dbmem")
				}
			}
			_ = AddLicense(t, client, lo)
		}
	}
	return client, provisionerCloser, coderAPI, user
}

// LicenseOptions is used to generate a license for testing.
// It supports the builder pattern for easy customization.
type LicenseOptions struct {
	AccountType   string
	AccountID     string
	DeploymentIDs []string
	Trial         bool
	FeatureSet    codersdk.FeatureSet
	AllFeatures   bool
	// GraceAt is the time at which the license will enter the grace period.
	GraceAt time.Time
	// ExpiresAt is the time at which the license will hard expire.
	// ExpiresAt should always be greater then GraceAt.
	ExpiresAt time.Time
	// NotBefore is the time at which the license becomes valid. If set to the
	// zero value, the `nbf` claim on the license is set to 1 minute in the
	// past.
	NotBefore time.Time
	Features  license.Features
}

func (opts *LicenseOptions) Expired(now time.Time) *LicenseOptions {
	opts.ExpiresAt = now.Add(time.Hour * 24 * -2)
	opts.GraceAt = now.Add(time.Hour * 24 * -3)
	return opts
}

func (opts *LicenseOptions) GracePeriod(now time.Time) *LicenseOptions {
	opts.ExpiresAt = now.Add(time.Hour * 24)
	opts.GraceAt = now.Add(time.Hour * 24 * -1)
	return opts
}

func (opts *LicenseOptions) Valid(now time.Time) *LicenseOptions {
	opts.ExpiresAt = now.Add(time.Hour * 24 * 60)
	opts.GraceAt = now.Add(time.Hour * 24 * 53)
	return opts
}

func (opts *LicenseOptions) FutureTerm(now time.Time) *LicenseOptions {
	opts.NotBefore = now.Add(time.Hour * 24)
	opts.ExpiresAt = now.Add(time.Hour * 24 * 60)
	opts.GraceAt = now.Add(time.Hour * 24 * 53)
	return opts
}

func (opts *LicenseOptions) UserLimit(limit int64) *LicenseOptions {
	return opts.Feature(codersdk.FeatureUserLimit, limit)
}

func (opts *LicenseOptions) Feature(name codersdk.FeatureName, value int64) *LicenseOptions {
	if opts.Features == nil {
		opts.Features = license.Features{}
	}
	opts.Features[name] = value
	return opts
}

func (opts *LicenseOptions) Generate(t *testing.T) string {
	return GenerateLicense(t, *opts)
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
	if options.NotBefore.IsZero() {
		options.NotBefore = time.Now().Add(-time.Minute)
	}

	c := &license.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    "test@testing.test",
			ExpiresAt: jwt.NewNumericDate(options.ExpiresAt),
			NotBefore: jwt.NewNumericDate(options.NotBefore),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
		LicenseExpires: jwt.NewNumericDate(options.GraceAt),
		AccountType:    options.AccountType,
		AccountID:      options.AccountID,
		DeploymentIDs:  options.DeploymentIDs,
		Trial:          options.Trial,
		Version:        license.CurrentVersion,
		AllFeatures:    options.AllFeatures,
		FeatureSet:     options.FeatureSet,
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

type CreateOrganizationOptions struct {
	// IncludeProvisionerDaemon will spin up an external provisioner for the organization.
	// This requires enterprise and the feature 'codersdk.FeatureExternalProvisionerDaemons'
	IncludeProvisionerDaemon bool
}

func CreateOrganization(t *testing.T, client *codersdk.Client, opts CreateOrganizationOptions, mutators ...func(*codersdk.CreateOrganizationRequest)) codersdk.Organization {
	ctx := testutil.Context(t, testutil.WaitMedium)
	req := codersdk.CreateOrganizationRequest{
		Name:        strings.ReplaceAll(strings.ToLower(namesgenerator.GetRandomName(0)), "_", "-"),
		DisplayName: namesgenerator.GetRandomName(1),
		Description: namesgenerator.GetRandomName(1),
		Icon:        "",
	}
	for _, mutator := range mutators {
		mutator(&req)
	}

	org, err := client.CreateOrganization(ctx, req)
	require.NoError(t, err)

	if opts.IncludeProvisionerDaemon {
		closer := NewExternalProvisionerDaemon(t, client, org.ID, map[string]string{})
		t.Cleanup(func() {
			_ = closer.Close()
		})
	}

	return org
}

func NewExternalProvisionerDaemon(t testing.TB, client *codersdk.Client, org uuid.UUID, tags map[string]string) io.Closer {
	t.Helper()

	// Without this check, the provisioner will silently fail.
	entitlements, err := client.Entitlements(context.Background())
	if err != nil {
		// AGPL instances will throw this error. They cannot use external
		// provisioners.
		t.Errorf("external provisioners requires a license with entitlements. The client failed to fetch the entitlements, is this an enterprise instance of coderd?")
		t.FailNow()
		return nil
	}

	feature := entitlements.Features[codersdk.FeatureExternalProvisionerDaemons]
	if !feature.Enabled || feature.Entitlement != codersdk.EntitlementEntitled {
		require.NoError(t, xerrors.Errorf("external provisioner daemons require an entitled license"))
		return nil
	}

	echoClient, echoServer := drpc.MemTransportPipe()
	ctx, cancelFunc := context.WithCancel(context.Background())
	serveDone := make(chan struct{})
	t.Cleanup(func() {
		_ = echoClient.Close()
		_ = echoServer.Close()
		cancelFunc()
		<-serveDone
	})
	go func() {
		defer close(serveDone)
		err := echo.Serve(ctx, &provisionersdk.ServeOptions{
			Listener:      echoServer,
			WorkDirectory: t.TempDir(),
		})
		assert.NoError(t, err)
	}()

	daemon := provisionerd.New(func(ctx context.Context) (provisionerdproto.DRPCProvisionerDaemonClient, error) {
		return client.ServeProvisionerDaemon(ctx, codersdk.ServeProvisionerDaemonRequest{
			ID:           uuid.New(),
			Name:         t.Name(),
			Organization: org,
			Provisioners: []codersdk.ProvisionerType{codersdk.ProvisionerTypeEcho},
			Tags:         tags,
		})
	}, &provisionerd.Options{
		Logger:              testutil.Logger(t).Named("provisionerd"),
		UpdateInterval:      250 * time.Millisecond,
		ForceCancelInterval: 5 * time.Second,
		Connector: provisionerd.LocalProvisioners{
			string(database.ProvisionerTypeEcho): sdkproto.NewDRPCProvisionerClient(echoClient),
		},
	})
	closer := coderdtest.NewProvisionerDaemonCloser(daemon)
	t.Cleanup(func() {
		_ = closer.Close()
	})

	return closer
}
