package server

import (
	"time"

	"github.com/go-oidfed/lib"
	"github.com/go-oidfed/lib/jwx"
	log "github.com/sirupsen/logrus"

	"github.com/zachmann/tip/internal/config"
)

var federationLeafEntity *oidfed.FederationLeaf

func initFederationEntity() error {
	fedConfig := config.Get().TIP.Federation

	if fedConfig.EntityID == "" {
		return nil
	}

	if err := initFederationKeys(); err != nil {
		return err
	}

	prMetadata := &oidfed.OAuthProtectedResourceMetadata{
		Resource:                          fedConfig.EntityID,
		ResourceSigningAlgValuesSupported: fedConfig.OIDC.Algs,
		ResourceName:                      fedConfig.ResourceName,
		ResourceDocumentation:             fedConfig.ResourceDocumentation,
		DisplayName:                       fedConfig.DisplayName,
		Description:                       fedConfig.Description,
		Keywords:                          fedConfig.Keywords,
		Contacts:                          fedConfig.Contacts,
		LogoURI:                           fedConfig.LogoURI,
		PolicyURI:                         fedConfig.PolicyURI,
		InformationURI:                    fedConfig.InformationURI,
		OrganizationName:                  fedConfig.OrganizationName,
		OrganizationURI:                   fedConfig.OrganizationURI,
		Extra:                             fedConfig.ExtraPRMetadata,
	}

	metadata := &oidfed.Metadata{
		OAuthProtectedResource: prMetadata,
	}

	lifetime := time.Duration(fedConfig.ConfigurationLifetime) * time.Second

	var err error
	federationLeafEntity, err = oidfed.NewFederationLeaf(
		fedConfig.EntityID,
		fedConfig.AuthorityHints,
		fedConfig.TrustAnchors,
		metadata,
		jwx.NewEntityStatementSigner(getFederationSigner()),
		lifetime,
		getOIDCSigner(),
		fedConfig.ExtraEntityConfigurationData,
	)
	if err != nil {
		return err
	}

	federationLeafEntity.FederationEntity = oidfed.DynamicFederationEntity{
		ID: federationLeafEntity.EntityID(),
		Metadata: func() (*oidfed.Metadata, error) {
			jwks, err := getOIDCSigner().JWKS()
			if err != nil {
				return nil, err
			}
			metadata.OAuthProtectedResource.JWKS = &jwks
			return metadata, nil
		},
		AuthorityHints: func() ([]string, error) {
			return fedConfig.AuthorityHints, nil
		},
		TrustAnchorHints: func() ([]string, error) {
			return fedConfig.TrustAnchors.EntityIDs(), nil
		},
		ConfigurationLifetime: func() (time.Duration, error) {
			return lifetime, nil
		},
		EntityStatementSigner: func() (*jwx.EntityStatementSigner, error) {
			return jwx.NewEntityStatementSigner(getFederationSigner()), nil
		},
		ShouldApplyInformationalClaims: func() (bool, error) {
			return fedConfig.PublishInformationalClaimsInFederationEntity, nil
		},
		Extra: func() (map[string]any, []string, error) {
			return fedConfig.ExtraEntityConfigurationData, nil, nil
		},
	}

	log.WithField("entity_id", fedConfig.EntityID).Info("Federation entity initialized")
	return nil
}
