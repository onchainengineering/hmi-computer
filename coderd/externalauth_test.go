package coderd_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/coderdtest/oidctest"
	"github.com/coder/coder/v2/coderd/database/dbtime"
	"github.com/coder/coder/v2/coderd/externalauth"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/codersdk/agentsdk"
	"github.com/coder/coder/v2/provisioner/echo"
	"github.com/coder/coder/v2/testutil"
)

func TestExternalAuthByID(t *testing.T) {
	t.Parallel()
	t.Run("Unauthenticated", func(t *testing.T) {
		t.Parallel()
		const providerID = "fake-github"
		fake := oidctest.NewFakeIDP(t, oidctest.WithServing())

		client := coderdtest.New(t, &coderdtest.Options{
			ExternalAuthConfigs: []*externalauth.Config{
				fake.ExternalAuthConfig(t, providerID, nil, func(cfg *externalauth.Config) {
					cfg.Type = codersdk.EnhancedExternalAuthProviderGitHub.String()
				}),
			},
		})
		coderdtest.CreateFirstUser(t, client)
		auth, err := client.ExternalAuthByID(context.Background(), providerID)
		require.NoError(t, err)
		require.False(t, auth.Authenticated)
	})
	t.Run("AuthenticatedNoUser", func(t *testing.T) {
		// Ensures that a provider that can't obtain a user can
		// still return that the provider is authenticated.
		t.Parallel()
		const providerID = "fake-azure"
		fake := oidctest.NewFakeIDP(t, oidctest.WithServing())

		client := coderdtest.New(t, &coderdtest.Options{
			ExternalAuthConfigs: []*externalauth.Config{
				// AzureDevops doesn't have a user endpoint!
				fake.ExternalAuthConfig(t, providerID, nil, func(cfg *externalauth.Config) {
					cfg.Type = codersdk.EnhancedExternalAuthProviderAzureDevops.String()
				}),
			},
		})

		coderdtest.CreateFirstUser(t, client)
		fake.ExternalLogin(t, client)

		auth, err := client.ExternalAuthByID(context.Background(), providerID)
		require.NoError(t, err)
		require.True(t, auth.Authenticated)
	})
	t.Run("AuthenticatedWithUser", func(t *testing.T) {
		t.Parallel()
		const providerID = "fake-github"
		fake := oidctest.NewFakeIDP(t, oidctest.WithServing())
		client := coderdtest.New(t, &coderdtest.Options{
			ExternalAuthConfigs: []*externalauth.Config{
				fake.ExternalAuthConfig(t, providerID, &oidctest.ExternalAuthConfigOptions{
					ValidatePayload: func(_ string) interface{} {
						return github.User{
							Login:     github.String("kyle"),
							AvatarURL: github.String("https://avatars.githubusercontent.com/u/12345678?v=4"),
						}
					},
				}, func(cfg *externalauth.Config) {
					cfg.Type = codersdk.EnhancedExternalAuthProviderGitHub.String()
				}),
			},
		})

		coderdtest.CreateFirstUser(t, client)
		// Login to external auth provider
		fake.ExternalLogin(t, client)

		auth, err := client.ExternalAuthByID(context.Background(), providerID)
		require.NoError(t, err)
		require.True(t, auth.Authenticated)
		require.NotNil(t, auth.User)
		require.Equal(t, "kyle", auth.User.Login)
	})
	t.Run("AuthenticatedWithInstalls", func(t *testing.T) {
		t.Parallel()
		const providerID = "fake-github"
		fake := oidctest.NewFakeIDP(t, oidctest.WithServing())

		// routes includes a route for /install that returns a list of installations
		routes := (&oidctest.ExternalAuthConfigOptions{
			ValidatePayload: func(_ string) interface{} {
				return github.User{
					Login:     github.String("kyle"),
					AvatarURL: github.String("https://avatars.githubusercontent.com/u/12345678?v=4"),
				}
			},
		}).AddRoute("/installs", func(_ string, rw http.ResponseWriter, r *http.Request) {
			httpapi.Write(r.Context(), rw, http.StatusOK, struct {
				Installations []github.Installation `json:"installations"`
			}{
				Installations: []github.Installation{{
					ID: github.Int64(12345678),
					Account: &github.User{
						Login: github.String("coder"),
					},
				}},
			})
		})
		client := coderdtest.New(t, &coderdtest.Options{
			ExternalAuthConfigs: []*externalauth.Config{
				fake.ExternalAuthConfig(t, providerID, routes, func(cfg *externalauth.Config) {
					cfg.AppInstallationsURL = cfg.ValidateURL + "/installs"
					cfg.Type = codersdk.EnhancedExternalAuthProviderGitHub.String()
				}),
			},
		})

		coderdtest.CreateFirstUser(t, client)
		fake.ExternalLogin(t, client)

		auth, err := client.ExternalAuthByID(context.Background(), providerID)
		require.NoError(t, err)
		require.True(t, auth.Authenticated)
		require.NotNil(t, auth.User)
		require.Equal(t, "kyle", auth.User.Login)
		require.NotNil(t, auth.AppInstallations)
		require.Len(t, auth.AppInstallations, 1)
	})
}

// TestExternalAuthManagement is for testing the apis interacting with
// external auths from the user perspective. We assume the external auth
// will always work, so we can test the managing apis like unlinking and
// listing.
func TestExternalAuthManagement(t *testing.T) {
	t.Parallel()
	t.Run("ListProviders", func(t *testing.T) {
		t.Parallel()
		const githubID = "fake-github"
		const gitlabID = "fake-gitlab"

		github := oidctest.NewFakeIDP(t, oidctest.WithServing())
		gitlab := oidctest.NewFakeIDP(t, oidctest.WithServing())

		owner := coderdtest.New(t, &coderdtest.Options{
			ExternalAuthConfigs: []*externalauth.Config{
				github.ExternalAuthConfig(t, githubID, nil, func(cfg *externalauth.Config) {
					cfg.Type = codersdk.EnhancedExternalAuthProviderGitHub.String()
				}),
				gitlab.ExternalAuthConfig(t, gitlabID, nil, func(cfg *externalauth.Config) {
					cfg.Type = codersdk.EnhancedExternalAuthProviderGitLab.String()
				}),
			},
		})
		ownerUser := coderdtest.CreateFirstUser(t, owner)
		// Just a regular user
		client, _ := coderdtest.CreateAnotherUser(t, owner, ownerUser.OrganizationID)
		ctx := testutil.Context(t, testutil.WaitLong)

		// List auths without any links.
		list, err := client.ListExternalAuths(ctx)
		require.NoError(t, err)
		require.Len(t, list.Providers, 2)
		require.Len(t, list.Links, 0)

		// Log into github
		github.ExternalLogin(t, client)

		list, err = client.ListExternalAuths(ctx)
		require.NoError(t, err)
		require.Len(t, list.Providers, 2)
		require.Len(t, list.Links, 1)
		require.Equal(t, list.Links[0].ProviderID, githubID)

		// Unlink
		err = client.UnlinkExternalAuthByID(ctx, githubID)
		require.NoError(t, err)

		list, err = client.ListExternalAuths(ctx)
		require.NoError(t, err)
		require.Len(t, list.Providers, 2)
		require.Len(t, list.Links, 0)
	})
}

func TestExternalAuthDevice(t *testing.T) {
	t.Parallel()
	t.Run("NotSupported", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{
			ExternalAuthConfigs: []*externalauth.Config{{
				ID: "test",
			}},
		})
		coderdtest.CreateFirstUser(t, client)
		_, err := client.ExternalAuthDeviceByID(context.Background(), "test")
		var sdkErr *codersdk.Error
		require.ErrorAs(t, err, &sdkErr)
		require.Equal(t, http.StatusBadRequest, sdkErr.StatusCode())
	})
	t.Run("FetchCode", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpapi.Write(r.Context(), w, http.StatusOK, codersdk.ExternalAuthDevice{
				UserCode: "hey",
			})
		}))
		defer srv.Close()
		client := coderdtest.New(t, &coderdtest.Options{
			ExternalAuthConfigs: []*externalauth.Config{{
				ID: "test",
				DeviceAuth: &externalauth.DeviceAuth{
					ClientID: "test",
					CodeURL:  srv.URL,
					Scopes:   []string{"repo"},
				},
			}},
		})
		coderdtest.CreateFirstUser(t, client)
		device, err := client.ExternalAuthDeviceByID(context.Background(), "test")
		require.NoError(t, err)
		require.Equal(t, "hey", device.UserCode)
	})
	t.Run("ExchangeCode", func(t *testing.T) {
		t.Parallel()
		resp := externalauth.ExchangeDeviceCodeResponse{
			Error: "authorization_pending",
		}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpapi.Write(r.Context(), w, http.StatusOK, resp)
		}))
		defer srv.Close()
		client := coderdtest.New(t, &coderdtest.Options{
			ExternalAuthConfigs: []*externalauth.Config{{
				ID: "test",
				DeviceAuth: &externalauth.DeviceAuth{
					ClientID: "test",
					TokenURL: srv.URL,
					Scopes:   []string{"repo"},
				},
			}},
		})
		coderdtest.CreateFirstUser(t, client)
		err := client.ExternalAuthDeviceExchange(context.Background(), "test", codersdk.ExternalAuthDeviceExchange{
			DeviceCode: "hey",
		})
		var sdkErr *codersdk.Error
		require.ErrorAs(t, err, &sdkErr)
		require.Equal(t, http.StatusBadRequest, sdkErr.StatusCode())
		require.Equal(t, "authorization_pending", sdkErr.Detail)

		resp = externalauth.ExchangeDeviceCodeResponse{
			AccessToken: "hey",
		}

		err = client.ExternalAuthDeviceExchange(context.Background(), "test", codersdk.ExternalAuthDeviceExchange{
			DeviceCode: "hey",
		})
		require.NoError(t, err)

		auth, err := client.ExternalAuthByID(context.Background(), "test")
		require.NoError(t, err)
		require.True(t, auth.Authenticated)
	})
	t.Run("TooManyRequests", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			// Github returns an html payload for this error.
			_, _ = w.Write([]byte(`Please wait a few minutes before you try again`))
		}))
		defer srv.Close()
		client := coderdtest.New(t, &coderdtest.Options{
			ExternalAuthConfigs: []*externalauth.Config{{
				ID: "test",
				DeviceAuth: &externalauth.DeviceAuth{
					ClientID: "test",
					CodeURL:  srv.URL,
					Scopes:   []string{"repo"},
				},
			}},
		})
		coderdtest.CreateFirstUser(t, client)
		_, err := client.ExternalAuthDeviceByID(context.Background(), "test")
		require.ErrorContains(t, err, "rate limit hit")
	})
}

// nolint:bodyclose
func TestExternalAuthCallback(t *testing.T) {
	t.Parallel()
	t.Run("NoMatchingConfig", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{
			IncludeProvisionerDaemon: true,
			ExternalAuthConfigs:      []*externalauth.Config{},
		})
		user := coderdtest.CreateFirstUser(t, client)
		authToken := uuid.NewString()
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionPlan:  echo.PlanComplete,
			ProvisionApply: echo.ProvisionApplyWithAgent(authToken),
		})
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, user.OrganizationID, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		agentClient := agentsdk.New(client.URL)
		agentClient.SetSessionToken(authToken)
		_, err := agentClient.ExternalAuth(context.Background(), agentsdk.ExternalAuthRequest{
			Match: "github.com",
		})
		var apiError *codersdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusNotFound, apiError.StatusCode())
	})
	t.Run("ReturnsURL", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{
			IncludeProvisionerDaemon: true,
			ExternalAuthConfigs: []*externalauth.Config{{
				InstrumentedOAuth2Config: &testutil.OAuth2Config{},
				ID:                       "github",
				Regex:                    regexp.MustCompile(`github\.com`),
				Type:                     codersdk.EnhancedExternalAuthProviderGitHub.String(),
			}},
		})
		user := coderdtest.CreateFirstUser(t, client)
		authToken := uuid.NewString()
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionPlan:  echo.PlanComplete,
			ProvisionApply: echo.ProvisionApplyWithAgent(authToken),
		})
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, user.OrganizationID, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		agentClient := agentsdk.New(client.URL)
		agentClient.SetSessionToken(authToken)
		token, err := agentClient.ExternalAuth(context.Background(), agentsdk.ExternalAuthRequest{
			Match: "github.com/asd/asd",
		})
		require.NoError(t, err)
		require.True(t, strings.HasSuffix(token.URL, fmt.Sprintf("/external-auth/%s", "github")), token.URL)
	})
	t.Run("UnauthorizedCallback", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{
			IncludeProvisionerDaemon: true,
			ExternalAuthConfigs: []*externalauth.Config{{
				InstrumentedOAuth2Config: &testutil.OAuth2Config{},
				ID:                       "github",
				Regex:                    regexp.MustCompile(`github\.com`),
				Type:                     codersdk.EnhancedExternalAuthProviderGitHub.String(),
			}},
		})
		resp := coderdtest.RequestExternalAuthCallback(t, "github", client)
		require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	})
	t.Run("AuthorizedCallback", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{
			IncludeProvisionerDaemon: true,
			ExternalAuthConfigs: []*externalauth.Config{{
				InstrumentedOAuth2Config: &testutil.OAuth2Config{},
				ID:                       "github",
				Regex:                    regexp.MustCompile(`github\.com`),
				Type:                     codersdk.EnhancedExternalAuthProviderGitHub.String(),
			}},
		})
		_ = coderdtest.CreateFirstUser(t, client)
		resp := coderdtest.RequestExternalAuthCallback(t, "github", client)
		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		location, err := resp.Location()
		require.NoError(t, err)
		require.Equal(t, "/external-auth/github", location.Path)

		// Callback again to simulate updating the token.
		resp = coderdtest.RequestExternalAuthCallback(t, "github", client)
		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	})
	t.Run("ValidateURL", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitLong)

		srv := httptest.NewServer(nil)
		defer srv.Close()
		client := coderdtest.New(t, &coderdtest.Options{
			IncludeProvisionerDaemon: true,
			ExternalAuthConfigs: []*externalauth.Config{{
				ValidateURL:              srv.URL,
				InstrumentedOAuth2Config: &testutil.OAuth2Config{},
				ID:                       "github",
				Regex:                    regexp.MustCompile(`github\.com`),
				Type:                     codersdk.EnhancedExternalAuthProviderGitHub.String(),
			}},
		})
		user := coderdtest.CreateFirstUser(t, client)
		authToken := uuid.NewString()
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionPlan:  echo.PlanComplete,
			ProvisionApply: echo.ProvisionApplyWithAgent(authToken),
		})
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, user.OrganizationID, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		agentClient := agentsdk.New(client.URL)
		agentClient.SetSessionToken(authToken)

		resp := coderdtest.RequestExternalAuthCallback(t, "github", client)
		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)

		// If the validation URL says unauthorized, the callback
		// URL to re-authenticate should be returned.
		srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		res, err := agentClient.ExternalAuth(ctx, agentsdk.ExternalAuthRequest{
			Match: "github.com/asd/asd",
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.URL)

		// If the validation URL gives a non-OK status code, this
		// should be treated as an internal server error.
		srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Something went wrong!"))
		})
		_, err = agentClient.ExternalAuth(ctx, agentsdk.ExternalAuthRequest{
			Match: "github.com/asd/asd",
		})
		var apiError *codersdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusInternalServerError, apiError.StatusCode())
		require.Equal(t, "validate external auth token: status 403: body: Something went wrong!", apiError.Detail)
	})

	t.Run("ExpiredNoRefresh", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{
			IncludeProvisionerDaemon: true,
			ExternalAuthConfigs: []*externalauth.Config{{
				InstrumentedOAuth2Config: &testutil.OAuth2Config{
					Token: &oauth2.Token{
						AccessToken:  "token",
						RefreshToken: "something",
						Expiry:       dbtime.Now().Add(-time.Hour),
					},
				},
				ID:        "github",
				Regex:     regexp.MustCompile(`github\.com`),
				Type:      codersdk.EnhancedExternalAuthProviderGitHub.String(),
				NoRefresh: true,
			}},
		})
		user := coderdtest.CreateFirstUser(t, client)
		authToken := uuid.NewString()
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionPlan:  echo.PlanComplete,
			ProvisionApply: echo.ProvisionApplyWithAgent(authToken),
		})
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, user.OrganizationID, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		agentClient := agentsdk.New(client.URL)
		agentClient.SetSessionToken(authToken)

		token, err := agentClient.ExternalAuth(context.Background(), agentsdk.ExternalAuthRequest{
			Match: "github.com/asd/asd",
		})
		require.NoError(t, err)
		require.NotEmpty(t, token.URL)

		// In the configuration, we set our OAuth provider
		// to return an expired token. Coder consumes this
		// and stores it.
		resp := coderdtest.RequestExternalAuthCallback(t, "github", client)
		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)

		// Because the token is expired and `NoRefresh` is specified,
		// a redirect URL should be returned again.
		token, err = agentClient.ExternalAuth(context.Background(), agentsdk.ExternalAuthRequest{
			Match: "github.com/asd/asd",
		})
		require.NoError(t, err)
		require.NotEmpty(t, token.URL)
	})

	t.Run("FullFlow", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, &coderdtest.Options{
			IncludeProvisionerDaemon: true,
			ExternalAuthConfigs: []*externalauth.Config{{
				InstrumentedOAuth2Config: &testutil.OAuth2Config{},
				ID:                       "github",
				Regex:                    regexp.MustCompile(`github\.com`),
				Type:                     codersdk.EnhancedExternalAuthProviderGitHub.String(),
			}},
		})
		user := coderdtest.CreateFirstUser(t, client)
		authToken := uuid.NewString()
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionPlan:  echo.PlanComplete,
			ProvisionApply: echo.ProvisionApplyWithAgent(authToken),
		})
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		coderdtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, user.OrganizationID, template.ID)
		coderdtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		agentClient := agentsdk.New(client.URL)
		agentClient.SetSessionToken(authToken)

		token, err := agentClient.ExternalAuth(context.Background(), agentsdk.ExternalAuthRequest{
			Match: "github.com/asd/asd",
		})
		require.NoError(t, err)
		require.NotEmpty(t, token.URL)

		// Start waiting for the token callback...
		tokenChan := make(chan agentsdk.ExternalAuthResponse, 1)
		go func() {
			token, err := agentClient.ExternalAuth(context.Background(), agentsdk.ExternalAuthRequest{
				Match:  "github.com/asd/asd",
				Listen: true,
			})
			assert.NoError(t, err)
			tokenChan <- token
		}()

		time.Sleep(250 * time.Millisecond)

		resp := coderdtest.RequestExternalAuthCallback(t, "github", client)
		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		token = <-tokenChan
		require.Equal(t, "access_token", token.Username)

		token, err = agentClient.ExternalAuth(context.Background(), agentsdk.ExternalAuthRequest{
			Match: "github.com/asd/asd",
		})
		require.NoError(t, err)
	})
}
