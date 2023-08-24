package oidctest

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coder/coder/v2/codersdk"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/go-jose/go-jose/v3"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/v2/coderd"
)

type FakeIDP struct {
	issuer   string
	key      *rsa.PrivateKey
	provider providerJSON
	handler  http.Handler
	cfg      *oauth2.Config

	// clientID to be used by coderd
	clientID     string
	clientSecret string
	logger       slog.Logger

	codeToStateMap *SyncMap[string, string]
	// Token -> Email
	accessTokens *SyncMap[string, string]
	// Refresh Token -> Email
	refreshTokensUsed    *SyncMap[string, bool]
	refreshTokens        *SyncMap[string, string]
	stateToIDTokenClaims *SyncMap[string, jwt.MapClaims]
	refreshIDTokenClaims *SyncMap[string, jwt.MapClaims]

	// hooks
	hookUserInfo      func(email string) jwt.MapClaims
	hookIDTokenClaims jwt.MapClaims
	fakeCoderd        func(req *http.Request) (*http.Response, error)
	// Optional if you want to use a real http network request assuming
	// it is not directed to the IDP.
	defaultClient *http.Client
	serve         bool
}

func WithLogging(t testing.TB, options *slogtest.Options) func(*FakeIDP) {
	return func(f *FakeIDP) {
		f.logger = slogtest.Make(t, options)
	}
}

func WithStaticUserInfo(info jwt.MapClaims) func(*FakeIDP) {
	return func(f *FakeIDP) {
		f.hookUserInfo = func(_ string) jwt.MapClaims {
			return info
		}
	}
}

func WithDynamicUserInfo(userInfoFunc func(email string) jwt.MapClaims) func(*FakeIDP) {
	return func(f *FakeIDP) {
		f.hookUserInfo = userInfoFunc
	}
}

func WithServing() func(*FakeIDP) {
	return func(f *FakeIDP) {
		f.serve = true
	}
}

const (
	authorizePath = "/oauth2/authorize"
	tokenPath     = "/oauth2/token"
	keysPath      = "/oauth2/keys"
	userInfoPath  = "/oauth2/userinfo"
)

func NewFakeIDP(t testing.TB, opts ...func(idp *FakeIDP)) *FakeIDP {
	t.Helper()

	block, _ := pem.Decode([]byte(testRSAPrivateKey))
	pkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	idp := &FakeIDP{
		key:                  pkey,
		clientID:             uuid.NewString(),
		clientSecret:         uuid.NewString(),
		logger:               slog.Make(),
		codeToStateMap:       NewSyncMap[string, string](),
		accessTokens:         NewSyncMap[string, string](),
		refreshTokens:        NewSyncMap[string, string](),
		refreshTokensUsed:    NewSyncMap[string, bool](),
		stateToIDTokenClaims: NewSyncMap[string, jwt.MapClaims](),
		refreshIDTokenClaims: NewSyncMap[string, jwt.MapClaims](),
		hookUserInfo:         func(email string) jwt.MapClaims { return jwt.MapClaims{} },
	}
	idp.handler = idp.httpHandler(t)

	for _, opt := range opts {
		opt(idp)
	}

	if idp.issuer == "" {
		idp.issuer = "https://coder.com"
	}

	idp.updateIssuerURL(t, idp.issuer)
	if idp.serve {
		idp.Serve(t)
	}

	return idp
}

func (f *FakeIDP) updateIssuerURL(t testing.TB, issuer string) {
	t.Helper()

	u, err := url.Parse(issuer)
	require.NoError(t, err, "invalid issuer URL")

	f.issuer = issuer
	// providerJSON is the JSON representation of the OpenID Connect provider
	// These are all the urls that the IDP will respond to.
	f.provider = providerJSON{
		Issuer:      issuer,
		AuthURL:     u.ResolveReference(&url.URL{Path: authorizePath}).String(),
		TokenURL:    u.ResolveReference(&url.URL{Path: tokenPath}).String(),
		JWKSURL:     u.ResolveReference(&url.URL{Path: keysPath}).String(),
		UserInfoURL: u.ResolveReference(&url.URL{Path: userInfoPath}).String(),
		Algorithms: []string{
			"RS256",
		},
	}
}

// Serve is optional, but turns the FakeIDP into a real http server.
func (f *FakeIDP) Serve(t testing.TB) *httptest.Server {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewUnstartedServer(f.handler)
	srv.Config.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	srv.Start()
	t.Cleanup(srv.CloseClientConnections)
	t.Cleanup(srv.Close)
	t.Cleanup(cancel)

	f.updateIssuerURL(t, srv.URL)
	return srv
}

// LoginClient does the full OIDC flow starting at the "LoginButton".
// The client argument is just to get the URL of the Coder instance.
func (f *FakeIDP) LoginClient(t testing.TB, client *codersdk.Client, idTokenClaims jwt.MapClaims) (*codersdk.Client, *http.Response) {
	t.Helper()

	coderOauthURL, err := client.URL.Parse("/api/v2/users/oidc/callback")
	require.NoError(t, err)
	f.SetRedirect(t, coderOauthURL.String())

	cli := f.HTTPClient(client.HTTPClient)
	shallowCpyCli := &(*cli)
	shallowCpyCli.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Store the idTokenClaims to the specific state request. This ties
		// the claims 1:1 with a given authentication flow.
		state := req.URL.Query().Get("state")
		f.stateToIDTokenClaims.Store(state, idTokenClaims)
		return nil
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", coderOauthURL.String(), nil)
	require.NoError(t, err)
	if shallowCpyCli.Jar == nil {
		shallowCpyCli.Jar, err = cookiejar.New(nil)
		require.NoError(t, err, "failed to create cookie jar")
	}

	res, err := shallowCpyCli.Do(req)
	require.NoError(t, err)

	// If the coder session token exists, return the new authed client!
	var user *codersdk.Client
	cookies := shallowCpyCli.Jar.Cookies(client.URL)
	for _, cookie := range cookies {
		if cookie.Name == codersdk.SessionTokenCookie {
			user = codersdk.New(client.URL)
			user.SetSessionToken(cookie.Value)
		}
	}

	t.Cleanup(func() {
		if res.Body != nil {
			res.Body.Close()
		}
	})
	return user, res
}

// OIDCCallback will emulate the IDP redirecting back to the Coder callback.
// This is helpful if no Coderd exists.
func (f *FakeIDP) OIDCCallback(t testing.TB, state string, idTokenClaims jwt.MapClaims) (*http.Response, error) {
	t.Helper()
	f.stateToIDTokenClaims.Store(state, idTokenClaims)

	baseCli := http.DefaultClient
	if f.fakeCoderd != nil {
		baseCli = &http.Client{
			Transport: fakeRoundTripper{
				roundTrip: func(req *http.Request) (*http.Response, error) {
					return f.fakeCoderd(req)
				},
			},
		}
	}

	cli := f.HTTPClient(baseCli)
	u := f.cfg.AuthCodeURL(state)
	req, err := http.NewRequest("GET", u, nil)
	require.NoError(t, err)

	resp, err := cli.Do(req.WithContext(context.Background()))
	require.NoError(t, err)

	t.Cleanup(func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	})
	return resp, nil
}

type providerJSON struct {
	Issuer      string   `json:"issuer"`
	AuthURL     string   `json:"authorization_endpoint"`
	TokenURL    string   `json:"token_endpoint"`
	JWKSURL     string   `json:"jwks_uri"`
	UserInfoURL string   `json:"userinfo_endpoint"`
	Algorithms  []string `json:"id_token_signing_alg_values_supported"`
}

// newCode enforces the code exchanged is actually a valid code
// created by the IDP.
func (f *FakeIDP) newCode(state string) string {
	code := uuid.NewString()
	f.codeToStateMap.Store(code, state)
	return code
}

// newToken enforces the access token exchanged is actually a valid access token
// created by the IDP.
func (f *FakeIDP) newToken(email string) string {
	accessToken := uuid.NewString()
	f.accessTokens.Store(accessToken, email)
	return accessToken
}

func (f *FakeIDP) newRefreshTokens(email string) string {
	refreshToken := uuid.NewString()
	f.refreshTokens.Store(refreshToken, email)
	return refreshToken
}

func (f *FakeIDP) authenticateBearerTokenRequest(t testing.TB, req *http.Request) (string, error) {
	t.Helper()

	auth := req.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	_, ok := f.accessTokens.Load(token)
	if !ok {
		return "", xerrors.New("invalid access token")
	}
	return token, nil
}

func (f *FakeIDP) authenticateOIDClientRequest(t testing.TB, req *http.Request) (url.Values, error) {
	t.Helper()

	data, _ := io.ReadAll(req.Body)
	values, err := url.ParseQuery(string(data))
	if !assert.NoError(t, err, "parse token request values") {
		return nil, xerrors.New("invalid token request")

	}

	if !assert.Equal(t, f.clientID, values.Get("client_id"), "client_id mismatch") {
		return nil, xerrors.New("client_id mismatch")
	}

	if !assert.Equal(t, f.clientSecret, values.Get("client_secret"), "client_secret mismatch") {
		return nil, xerrors.New("client_secret mismatch")
	}

	return values, nil
}

func (f *FakeIDP) encodeClaims(t testing.TB, claims jwt.MapClaims) string {
	t.Helper()

	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(time.Hour).UnixMilli()
	}

	if _, ok := claims["aud"]; !ok {
		claims["aud"] = f.clientID
	}

	if _, ok := claims["iss"]; !ok {
		claims["iss"] = f.issuer
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(f.key)
	require.NoError(t, err)

	return signed
}

func (f *FakeIDP) httpHandler(t testing.TB) http.Handler {
	t.Helper()

	mux := chi.NewMux()
	// This endpoint is required to initialize the OIDC provider.
	// It is used to get the OIDC configuration.
	mux.Get("/.well-known/openid-configuration", func(rw http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(rw).Encode(f.provider)
	})

	// Authorize is called when the user is redirected to the IDP to login.
	// This is the browser hitting the IDP and the user logging into Google or
	// w/e and clicking "Allow". They will be redirected back to the redirect
	// when this is done.
	mux.Handle(authorizePath, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		f.logger.Info(r.Context(), "HTTP Call Authorize", slog.F("url", string(r.URL.String())))

		clientID := r.URL.Query().Get("client_id")
		if clientID != f.clientID {
			t.Errorf("unexpected client_id %q", clientID)
			http.Error(rw, "invalid client_id", http.StatusBadRequest)
		}

		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")

		scope := r.URL.Query().Get("scope")
		var _ = scope

		responseType := r.URL.Query().Get("response_type")
		switch responseType {
		case "code":
		case "token":
			t.Errorf("response_type %q not supported", responseType)
			http.Error(rw, "invalid response_type", http.StatusBadRequest)
			return
		default:
			t.Errorf("unexpected response_type %q", responseType)
			http.Error(rw, "invalid response_type", http.StatusBadRequest)
			return
		}

		ru, err := url.Parse(redirectURI)
		if err != nil {
			t.Errorf("invalid redirect_uri %q", redirectURI)
			http.Error(rw, "invalid redirect_uri", http.StatusBadRequest)
			return
		}

		q := ru.Query()
		q.Set("state", state)
		q.Set("code", f.newCode(state))
		ru.RawQuery = q.Encode()

		http.Redirect(rw, r, ru.String(), http.StatusTemporaryRedirect)
	}))

	mux.Handle(tokenPath, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		values, err := f.authenticateOIDClientRequest(t, r)
		f.logger.Info(r.Context(), "HTTP Call Token",
			slog.Error(err),
			slog.F("values", values.Encode()),
		)
		if err != nil {
			http.Error(rw, fmt.Sprintf("invalid token request: %s", err.Error()), http.StatusBadRequest)
			return
		}

		var claims jwt.MapClaims
		switch values.Get("grant_type") {
		case "authorization_code":
			code := values.Get("code")
			if !assert.NotEmpty(t, code, "code is empty") {
				http.Error(rw, "invalid code", http.StatusBadRequest)
				return
			}
			stateStr, ok := f.codeToStateMap.Load(code)
			if !assert.True(t, ok, "invalid code") {
				http.Error(rw, "invalid code", http.StatusBadRequest)
				return
			}
			// Always invalidate the code after it is used.
			f.codeToStateMap.Delete(code)

			idTokenClaims, ok := f.stateToIDTokenClaims.Load(stateStr)
			if !ok {
				t.Errorf("missing id token claims")
				http.Error(rw, "missing id token claims", http.StatusBadRequest)
				return
			}
			claims = idTokenClaims
		case "refresh_token":
			refreshToken := values.Get("refresh_token")
			if !assert.NotEmpty(t, refreshToken, "refresh_token is empty") {
				http.Error(rw, "invalid refresh_token", http.StatusBadRequest)
				return
			}

			_, ok := f.refreshTokens.Load(refreshToken)
			if !assert.True(t, ok, "invalid refresh_token") {
				http.Error(rw, "invalid refresh_token", http.StatusBadRequest)
				return
			}
			// Always invalidate the refresh token after it is used.
			f.refreshTokens.Delete(refreshToken)

			idTokenClaims, ok := f.refreshIDTokenClaims.Load(refreshToken)
			if !ok {
				t.Errorf("missing id token claims in refresh")
				http.Error(rw, "missing id token claims in refresh", http.StatusBadRequest)
				return
			}
			claims = idTokenClaims
			f.refreshTokensUsed.Store(refreshToken, true)
		default:
			t.Errorf("unexpected grant_type %q", values.Get("grant_type"))
			http.Error(rw, "invalid grant_type", http.StatusBadRequest)
			return
		}

		exp := time.Now().Add(time.Minute * 5)
		claims["exp"] = exp.UnixMilli()
		email, ok := claims["email"]
		if !ok || email.(string) == "" {
			email = "unknown"
		}
		refreshToken := f.newRefreshTokens(email.(string))
		token := map[string]interface{}{
			"access_token":  f.newToken(email.(string)),
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    int64(time.Minute * 5),
			"expiry":        exp.Unix(),
			"id_token":      f.encodeClaims(t, claims),
		}
		// Store the claims for the next refresh
		f.refreshIDTokenClaims.Store(refreshToken, claims)

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(token)
	}))

	mux.Handle(userInfoPath, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		token, err := f.authenticateBearerTokenRequest(t, r)
		f.logger.Info(r.Context(), "HTTP Call UserInfo",
			slog.Error(err),
		)
		if err != nil {
			http.Error(rw, fmt.Sprintf("invalid user info request: %s", err.Error()), http.StatusBadRequest)
			return
		}
		var _ = token

		email, ok := f.accessTokens.Load(token)
		if !ok {
			t.Errorf("access token user for user_info has no email to indicate which user")
			http.Error(rw, "invalid access token, missing user info", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(rw).Encode(f.hookUserInfo(email))
	}))

	mux.Handle(keysPath, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		f.logger.Info(r.Context(), "HTTP Call Keys")
		set := jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{
				{
					Key:       f.key.Public(),
					KeyID:     "test-key",
					Algorithm: "RSA",
				},
			},
		}
		_ = json.NewEncoder(rw).Encode(set)
	}))

	mux.NotFound(func(rw http.ResponseWriter, r *http.Request) {
		f.logger.Error(r.Context(), "HTTP Call NotFound", slog.F("path", r.URL.Path))
		t.Errorf("unexpected request to IDP at path %q. Not supported", r.URL.Path)
	})

	return mux
}

// HTTPClient runs the IDP in memory and returns an http.Client that can be used
// to make requests to the IDP. All requests are handled in memory, and no network
// requests are made.
//
// If a request is not to the IDP, then the passed in client will be used.
// If no client is passed in, then any regular network requests will fail.
func (f *FakeIDP) HTTPClient(rest *http.Client) *http.Client {
	if f.serve {
		if rest == nil {
			return http.DefaultClient
		}
		return rest
	}
	return &http.Client{
		Transport: fakeRoundTripper{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				u, _ := url.Parse(f.issuer)
				if req.URL.Host != u.Host {
					if rest == nil {
						return nil, fmt.Errorf("unexpected network request to %q", req.URL.Host)
					}
					return rest.Transport.RoundTrip(req)
				}
				resp := httptest.NewRecorder()
				f.handler.ServeHTTP(resp, req)
				return resp.Result(), nil
			},
		},
	}
}

// RefreshUsed returns if the refresh token has been used. All refresh tokens
// can only be used once, then they are deleted.
func (f *FakeIDP) RefreshUsed(refreshToken string) bool {
	used, _ := f.refreshTokensUsed.Load(refreshToken)
	return used
}

func (f *FakeIDP) UpdateRefreshClaims(refreshToken string, claims jwt.MapClaims) {
	f.refreshIDTokenClaims.Store(refreshToken, claims)
}

func (f *FakeIDP) SetRedirect(t testing.TB, url string) {
	t.Helper()

	f.cfg.RedirectURL = url
}

func (f *FakeIDP) SetCoderdCallback(callback func(req *http.Request) (*http.Response, error)) {
	f.fakeCoderd = callback
}

func (f *FakeIDP) SetCoderdCallbackHandler(handler http.HandlerFunc) {
	if f.serve {
		panic("cannot set callback handler when using 'WithServing'. Must implement an actual 'Coderd'")
	}
	f.fakeCoderd = func(req *http.Request) (*http.Response, error) {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		return resp.Result(), nil
	}
}

func (f *FakeIDP) OIDCConfig(t testing.TB, scopes []string, opts ...func(cfg *coderd.OIDCConfig)) *coderd.OIDCConfig {
	t.Helper()
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	oauthCfg := &oauth2.Config{
		ClientID:     f.clientID,
		ClientSecret: f.clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   f.provider.AuthURL,
			TokenURL:  f.provider.TokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		// If the user is using a real network request, they will need to do
		// 'fake.SetRedirect()'
		RedirectURL: "https://redirect.com",
		Scopes:      scopes,
	}

	ctx := oidc.ClientContext(context.Background(), f.HTTPClient(nil))
	p, err := oidc.NewProvider(ctx, f.provider.Issuer)
	require.NoError(t, err, "failed to create OIDC provider")
	cfg := &coderd.OIDCConfig{
		OAuth2Config: oauthCfg,
		Provider:     p,
		Verifier: oidc.NewVerifier(f.provider.Issuer, &oidc.StaticKeySet{
			PublicKeys: []crypto.PublicKey{f.key.Public()},
		}, &oidc.Config{
			ClientID: oauthCfg.ClientID,
			SupportedSigningAlgs: []string{
				"RS256",
			},
			// Todo: add support for Now()
		}),
		UsernameField: "preferred_username",
		EmailField:    "email",
		AuthURLParams: map[string]string{"access_type": "offline"},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	f.cfg = oauthCfg

	return cfg
}

type fakeRoundTripper struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

func (f fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f.roundTrip(req)
}

const testRSAPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDLets8+7M+iAQAqN/5BVyCIjhTQ4cmXulL+gm3v0oGMWzLupUS
v8KPA+Tp7dgC/DZPfMLaNH1obBBhJ9DhS6RdS3AS3kzeFrdu8zFHLWF53DUBhS92
5dCAEuJpDnNizdEhxTfoHrhuCmz8l2nt1pe5eUK2XWgd08Uc93h5ij098wIDAQAB
AoGAHLaZeWGLSaen6O/rqxg2laZ+jEFbMO7zvOTruiIkL/uJfrY1kw+8RLIn+1q0
wLcWcuEIHgKKL9IP/aXAtAoYh1FBvRPLkovF1NZB0Je/+CSGka6wvc3TGdvppZJe
rKNcUvuOYLxkmLy4g9zuY5qrxFyhtIn2qZzXEtLaVOHzPQECQQDvN0mSajpU7dTB
w4jwx7IRXGSSx65c+AsHSc1Rj++9qtPC6WsFgAfFN2CEmqhMbEUVGPv/aPjdyWk9
pyLE9xR/AkEA2cGwyIunijE5v2rlZAD7C4vRgdcMyCf3uuPcgzFtsR6ZhyQSgLZ8
YRPuvwm4cdPJMmO3YwBfxT6XGuSc2k8MjQJBAI0+b8prvpV2+DCQa8L/pjxp+VhR
Xrq2GozrHrgR7NRokTB88hwFRJFF6U9iogy9wOx8HA7qxEbwLZuhm/4AhbECQC2a
d8h4Ht09E+f3nhTEc87mODkl7WJZpHL6V2sORfeq/eIkds+H6CJ4hy5w/bSw8tjf
sz9Di8sGIaUbLZI2rd0CQQCzlVwEtRtoNCyMJTTrkgUuNufLP19RZ5FpyXxBO5/u
QastnN77KfUwdj3SJt44U/uh1jAIv4oSLBr8HYUkbnI8
-----END RSA PRIVATE KEY-----`
