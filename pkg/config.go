package pkg

import (
	"github.com/oidc-mytoken/utils/httpclient"
	"github.com/oidc-mytoken/utils/utils"
	log "github.com/sirupsen/logrus"
)

type TIPConfig struct {
	LinkedIssuer   linkedIssuerConf   `yaml:"linked_issuer"`
	RemoteIssuers  []remoteIssuerConf `yaml:"remote_issuers"`
	FallbackIssuer remoteIssuerConf   `yaml:"fallback_issuer"`
	Federation     federationConf     `yaml:"federation"`
}

type linkedIssuerConf struct {
	IssuerURL                   string `yaml:"issuer_url"`
	NativeIntrospectionEndpoint string `yaml:"native_introspection_endpoint"`
}

type remoteIssuerConf struct {
	IssuerURL             string            `yaml:"issuer_url"`
	IntrospectionEndpoint string            `yaml:"introspection_endpoint"`
	ClientID              string            `yaml:"client_id"`
	ClientSecret          string            `yaml:"client_secret"`
	DropClaims            []string          `yaml:"drop_claims"`
	ClaimMapping          claimsMapping     `yaml:"claim_mapping"`
	ClaimRenaming         map[string]string `yaml:"claim_renaming"`
}

func (ric *remoteIssuerConf) discoverEndpoint() {
	if ric.IntrospectionEndpoint != "" {
		return
	}
	configEndpoints := []string{
		utils.CombineURLPath(ric.IssuerURL, ".well-known/openid-configuration"),
		utils.CombineURLPath(ric.IssuerURL, ".well-known/oauth-authorization-server"),
	}
	for _, endpoint := range configEndpoints {
		var metadata struct {
			IntrospectionEndpoint string `json:"introspection_endpoint"`
		}
		_, err := httpclient.Do().R().
			SetResult(&metadata).
			Get(endpoint)
		if err != nil {
			log.WithError(err).Warn("Failed to discover endpoint")
			continue
		}
		ric.IntrospectionEndpoint = metadata.IntrospectionEndpoint
		if ric.IntrospectionEndpoint != "" {
			break
		}
	}
	if ric.IntrospectionEndpoint == "" {
		log.Fatal("could not obtain introspection endpoint")
	}
}

type claimsMapping struct {
	Strings      stringsClaimsMapping      `yaml:"strings"`
	StringSlices stringSlicesClaimsMapping `yaml:"string_arrays"`
}
type stringsClaimsMapping map[string]map[string]string
type stringSlicesClaimsMapping map[string]map[string][]string

type federationConf struct {
}

func (m stringsClaimsMapping) translate(claim, value string) string {
	cm, found := m[claim]
	if !found {
		return value
	}
	transformed, ok := cm[value]
	if ok {
		return transformed
	}
	return value
}

func (m stringSlicesClaimsMapping) translate(claim string, values []string) (final []string) {
	cm, found := m[claim]
	if !found {
		return values
	}
	for _, v := range values {
		transformed, ok := cm[v]
		if ok {
			final = append(final, transformed...)
		} else {
			final = append(final, v)
		}
	}
	return
}
