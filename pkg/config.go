package pkg

import (
	"fmt"
	"net/url"
	"path/filepath"

	oidfed "github.com/go-oidfed/lib"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/oidc-mytoken/utils/httpclient"
	"github.com/oidc-mytoken/utils/utils"
	"github.com/oidc-mytoken/utils/utils/fileutil"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type TIPConfig struct {
	LinkedIssuer              LinkedIssuerConf   `yaml:"linked_issuer"`
	RemoteIssuers             []remoteIssuerConf `yaml:"remote_issuers"`
	FallbackIssuerUnknown     remoteIssuerConf   `yaml:"fallback_issuer_unknown_token_issuer"`
	FallbackIssuerUnsupported remoteIssuerConf   `yaml:"fallback_issuer_unsupported_token_issuer"`
	Federation                FederationConf     `yaml:"federation"`
}

type LinkedIssuerConf struct {
	IssuerURL                   string `yaml:"issuer_url"`
	NativeIntrospectionEndpoint string `yaml:"native_introspection_endpoint"`
	ProxyWellKnown              bool   `yaml:"proxy_well_known"`
	PublicIntrospectionEndpoint string `yaml:"public_introspection_endpoint"`
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

type FederationConf struct {
	EntityID       string              `yaml:"entity_id"`
	TrustAnchors   oidfed.TrustAnchors `yaml:"trust_anchors"`
	AuthorityHints []string            `yaml:"authority_hints"`

	KeyDir string `yaml:"key_dir"`

	Federation SigningConf `yaml:"federation"`
	OIDC       SigningConf `yaml:"oidc"`

	ResourceName          string `yaml:"resource_name"`
	ResourceDocumentation string `yaml:"resource_documentation"`

	DisplayName      string   `yaml:"display_name"`
	Description      string   `yaml:"description"`
	Keywords         []string `yaml:"keywords"`
	Contacts         []string `yaml:"contacts"`
	LogoURI          string   `yaml:"logo_uri"`
	PolicyURI        string   `yaml:"policy_uri"`
	InformationURI   string   `yaml:"information_uri"`
	OrganizationName string   `yaml:"organization_name"`
	OrganizationURI  string   `yaml:"organization_uri"`

	ExtraPRMetadata              map[string]any `yaml:"extra_pr_metadata"`
	ExtraFEMetadata              map[string]any `yaml:"extra_fe_metadata"`
	ExtraEntityConfigurationData map[string]any `yaml:"extra_entity_configuration_data"`

	PublishInformationalClaimsInFederationEntity bool `yaml:"publish_informational_claims_in_federation_entity"`

	ConfigurationLifetime int `yaml:"configuration_lifetime"`
}

type SigningConf struct {
	Alg          string          `yaml:"alg"`
	Algs         []string        `yaml:"algs"`
	RSAKeyLen    int             `yaml:"rsa_key_len"`
	GenerateKeys bool            `yaml:"generate_keys"`
	KeyRotation  keyRotationConf `yaml:"automatic_key_rollover"`
}

type keyRotationConf struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"`
	Overlap  int  `yaml:"overlap"`
}

func (sc *SigningConf) normalize() {
	if sc.Alg != "" {
		if len(sc.Algs) == 0 {
			sc.Algs = []string{sc.Alg}
		}
	}
	if sc.RSAKeyLen == 0 {
		sc.RSAKeyLen = 2048
	}
}

func (fc *FederationConf) Validate() error {
	if fc.EntityID == "" {
		return nil
	}

	if len(fc.TrustAnchors) == 0 {
		return errors.New("federation.trust_anchors is required when entity_id is set")
	}

	if fc.KeyDir == "" {
		return errors.New("federation.key_dir is required when entity_id is set")
	}

	if !fileutil.FileExists(fc.KeyDir) {
		return errors.Errorf("federation.key_dir '%s' does not exist", fc.KeyDir)
	}

	fc.Federation.normalize()
	fc.OIDC.normalize()

	if fc.Federation.Alg == "" {
		fc.Federation.Alg = "ES512"
	}
	if len(fc.Federation.Algs) == 0 {
		fc.Federation.Algs = []string{fc.Federation.Alg}
	}

	if len(fc.Federation.Algs) != 1 {
		return errors.New("federation.federation.algs must contain exactly one algorithm")
	}

	fedAlg, ok := jwa.LookupSignatureAlgorithm(fc.Federation.Algs[0])
	if !ok {
		return errors.Errorf("federation.federation.alg '%s' is not supported", fc.Federation.Algs[0])
	}

	for _, algStr := range fc.OIDC.Algs {
		if _, ok := jwa.LookupSignatureAlgorithm(algStr); !ok {
			return errors.Errorf("federation.oidc.alg '%s' is not supported", algStr)
		}
	}

	if fc.ConfigurationLifetime == 0 {
		fc.ConfigurationLifetime = 86400
	}

	for i, hint := range fc.AuthorityHints {
		if _, err := url.Parse(hint); err != nil {
			return errors.Errorf("federation.authority_hints[%d] '%s' is not a valid URL: %v", i, hint, err)
		}
	}

	if !fc.Federation.GenerateKeys {
		keyFile := filepath.Join(fc.KeyDir, fmt.Sprintf("%s.pem", fedAlg.String()))
		if !fileutil.FileExists(keyFile) {
			return errors.Errorf(
				"federation.federation.generate_keys is false but key file '%s' does not exist", keyFile,
			)
		}
	}

	if !fc.OIDC.GenerateKeys {
		for _, algStr := range fc.OIDC.Algs {
			alg, _ := jwa.LookupSignatureAlgorithm(algStr)
			keyFile := filepath.Join(fc.KeyDir, fmt.Sprintf("%s_%s.pem", "oidc", alg.String()))
			if !fileutil.FileExists(keyFile) {
				return errors.Errorf("federation.oidc.generate_keys is false but key file '%s' does not exist", keyFile)
			}
		}
	}

	return nil
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
