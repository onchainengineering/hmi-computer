package trialer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onchainengineering/hmi-computer/v2/coderd/database/dbmem"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/codersdk"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/enterprise/coderd/coderdenttest"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/enterprise/trialer"
)

func TestTrialer(t *testing.T) {
	t.Parallel()
	license := coderdenttest.GenerateLicense(t, coderdenttest.LicenseOptions{
		Trial: true,
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(license))
	}))
	defer srv.Close()
	db := dbmem.New()

	gen := trialer.New(db, srv.URL, coderdenttest.Keys)
	err := gen(context.Background(), codersdk.LicensorTrialRequest{Email: "kyle+colin@coder.com"})
	require.NoError(t, err)
	licenses, err := db.GetLicenses(context.Background())
	require.NoError(t, err)
	require.Len(t, licenses, 1)
}
