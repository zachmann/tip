package pkg

import (
	"sync"
	"time"

	"github.com/oidc-mytoken/utils/httpclient"
	"github.com/oidc-mytoken/utils/utils"
	"github.com/pkg/errors"
)

const defaultCacheTTL = 1 * time.Hour

// WellKnownProxy handles fetching and caching the OpenID configuration metadata
// from the linked issuer, replacing the introspection_endpoint with TIP's endpoint.
type WellKnownProxy struct {
	conf       LinkedIssuerConf
	cachedData map[string]any
	cachedAt   time.Time
	cacheTTL   time.Duration
	mu         sync.RWMutex
}

// NewWellKnownProxy creates a new WellKnownProxy for the given linked issuer configuration.
func NewWellKnownProxy(conf LinkedIssuerConf) *WellKnownProxy {
	return &WellKnownProxy{
		conf:     conf,
		cacheTTL: defaultCacheTTL,
	}
}

// GetMetadata returns the OpenID configuration metadata with the introspection_endpoint
// replaced with TIP's public endpoint. Results are cached for the configured TTL.
func (w *WellKnownProxy) GetMetadata() (map[string]any, error) {
	w.mu.RLock()
	if w.cachedData != nil && time.Since(w.cachedAt) < w.cacheTTL {
		data := w.cachedData
		w.mu.RUnlock()
		return data, nil
	}
	w.mu.RUnlock()

	// Cache miss or expired, fetch fresh data
	return w.fetchAndCache()
}

func (w *WellKnownProxy) fetchAndCache() (map[string]any, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Double-check after acquiring write lock
	if w.cachedData != nil && time.Since(w.cachedAt) < w.cacheTTL {
		return w.cachedData, nil
	}

	endpoint := utils.CombineURLPath(w.conf.IssuerURL, ".well-known/openid-configuration")

	var metadata map[string]any
	_, err := httpclient.Do().R().
		SetResult(&metadata).
		Get(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch openid-configuration from linked issuer")
	}

	if metadata == nil {
		return nil, errors.New("received empty metadata from linked issuer")
	}

	// Replace introspection_endpoint with TIP's public endpoint
	metadata["introspection_endpoint"] = w.conf.PublicIntrospectionEndpoint

	w.cachedData = metadata
	w.cachedAt = time.Now()

	return metadata, nil
}
