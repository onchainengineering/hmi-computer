package wsproxy

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/onchainengineering/hmi-computer/v2/coderd/cryptokeys"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/codersdk"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/enterprise/wsproxy/wsproxysdk"
)

var _ cryptokeys.Fetcher = &ProxyFetcher{}

type ProxyFetcher struct {
	Client *wsproxysdk.Client
}

func (p *ProxyFetcher) Fetch(ctx context.Context, feature codersdk.CryptoKeyFeature) ([]codersdk.CryptoKey, error) {
	keys, err := p.Client.CryptoKeys(ctx, feature)
	if err != nil {
		return nil, xerrors.Errorf("crypto keys: %w", err)
	}
	return keys.CryptoKeys, nil
}
