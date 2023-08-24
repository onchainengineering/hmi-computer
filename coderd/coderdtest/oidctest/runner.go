package oidctest

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/testutil"
)

// LoginHelper helps with logging in a user and refreshing their oauth tokens.
// It is mainly because refreshing oauth tokens is a bit tricky and requires
// some database manipulation.
type LoginHelper struct {
	fake  *FakeIDP
	owner *codersdk.Client
}

func NewLoginHelper(owner *codersdk.Client, fake *FakeIDP) *LoginHelper {
	if owner == nil {
		panic("owner must not be nil")
	}
	if fake == nil {
		panic("fake must not be nil")
	}
	return &LoginHelper{
		fake:  fake,
		owner: owner,
	}
}

// Login just helps by making an unauthenticated client and logging in with
// the given claims. All Logins should be unauthenticated, so this is a
// convenience method.
func (h *LoginHelper) Login(t *testing.T, idTokenClaims jwt.MapClaims) (*codersdk.Client, *http.Response) {
	t.Helper()
	unauthenticatedClient := codersdk.New(h.owner.URL)

	return h.fake.LoginClient(t, unauthenticatedClient, idTokenClaims)
}

// ForceRefresh forces the client to refresh its oauth token.
func (h *LoginHelper) ForceRefresh(t *testing.T, db database.Store, user *codersdk.Client, idToken jwt.MapClaims) (authenticatedCall func(t *testing.T)) {
	t.Helper()

	//nolint:gocritic // Testing
	ctx := dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitMedium))

	id, _, err := httpmw.SplitAPIToken(user.SessionToken())
	require.NoError(t, err)

	// We need to get the OIDC link and update it in the database to force
	// it to be expired.
	key, err := db.GetAPIKeyByID(ctx, id)
	require.NoError(t, err, "get api key")

	link, err := db.GetUserLinkByUserIDLoginType(ctx, database.GetUserLinkByUserIDLoginTypeParams{
		UserID:    key.UserID,
		LoginType: database.LoginTypeOIDC,
	})
	require.NoError(t, err, "get user link")

	// Updates the claims that the IDP will return. By default, it always
	// uses the original claims for the original oauth token.
	h.fake.UpdateRefreshClaims(link.OAuthRefreshToken, idToken)

	// Fetch the oauth link for the given user.
	_, err = db.UpdateUserLink(ctx, database.UpdateUserLinkParams{
		OAuthAccessToken:  link.OAuthAccessToken,
		OAuthRefreshToken: link.OAuthRefreshToken,
		OAuthExpiry:       time.Now().Add(time.Hour * -1),
		UserID:            key.UserID,
		LoginType:         database.LoginTypeOIDC,
	})
	require.NoError(t, err, "expire user link")
	t.Cleanup(func() {
		require.True(t, h.fake.RefreshUsed(link.OAuthRefreshToken), "refresh token must be used, but has not. Did you forget to call the returned function from this call?")
	})

	return func(t *testing.T) {
		t.Helper()

		// Do any authenticated call to force the refresh
		_, err := user.User(testutil.Context(t, testutil.WaitShort), "me")
		require.NoError(t, err, "user must be able to be fetched")
	}
}
