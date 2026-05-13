package server

import (
	"fmt"
	"time"

	"github.com/go-oidfed/lib/jwx"
	"github.com/go-oidfed/lib/jwx/keymanagement/kms"
	"github.com/go-oidfed/lib/jwx/keymanagement/public"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/zachmann/go-utils/duration"

	"github.com/zachmann/tip/internal/config"
)

var (
	oidcSigner       jwx.VersatileSigner
	federationSigner jwx.VersatileSigner
)

func initFederationKeys() error {
	conf := config.Get().TIP.Federation

	fedAlg, ok := jwa.LookupSignatureAlgorithm(conf.Federation.Algs[0])
	if !ok {
		return fmt.Errorf("invalid federation signing algorithm: %s", conf.Federation.Algs[0])
	}

	var oidcAlgs []jwa.SignatureAlgorithm
	for _, algStr := range conf.OIDC.Algs {
		alg, ok := jwa.LookupSignatureAlgorithm(algStr)
		if !ok {
			return fmt.Errorf("invalid OIDC signing algorithm: %s", algStr)
		}
		oidcAlgs = append(oidcAlgs, alg)
	}

	federationPks := &public.FilesystemPublicKeyStorage{
		Dir:    conf.KeyDir,
		TypeID: "federation",
	}
	if err := federationPks.Load(); err != nil {
		return err
	}

	fedKmsConfig := kms.KMSConfig{
		GenerateKeys: conf.Federation.GenerateKeys,
		DefaultAlg:   fedAlg,
		RSAKeyLen:    conf.Federation.RSAKeyLen,
		KeyRotation: kms.KeyRotationConfig{
			Enabled:  conf.Federation.KeyRotation.Enabled,
			Interval: duration.DurationOption(time.Duration(conf.Federation.KeyRotation.Interval) * time.Second),
			Overlap:  duration.DurationOption(time.Duration(conf.Federation.KeyRotation.Overlap) * time.Second),
		},
	}

	federationKMS := kms.NewSingleAlgFilesystemKMS(
		fedAlg,
		kms.FilesystemKMSConfig{
			KMSConfig: fedKmsConfig,
			Dir:       conf.KeyDir,
			TypeID:    "federation",
		},
		federationPks,
	)
	if err := federationKMS.Load(); err != nil {
		return err
	}

	federationSigner = kms.KMSToVersatileSignerWithPKStorage(federationKMS, federationPks)

	oidcPks := &public.FilesystemPublicKeyStorage{
		Dir:    conf.KeyDir,
		TypeID: "oidc",
	}
	if err := oidcPks.Load(); err != nil {
		return err
	}

	oidcKmsConfig := kms.KMSConfig{
		GenerateKeys: conf.OIDC.GenerateKeys,
		Algs:         oidcAlgs,
		DefaultAlg:   oidcAlgs[0],
		RSAKeyLen:    conf.OIDC.RSAKeyLen,
		KeyRotation: kms.KeyRotationConfig{
			Enabled:  conf.OIDC.KeyRotation.Enabled,
			Interval: duration.DurationOption(time.Duration(conf.OIDC.KeyRotation.Interval) * time.Second),
			Overlap:  duration.DurationOption(time.Duration(conf.OIDC.KeyRotation.Overlap) * time.Second),
		},
	}

	oidcKMS, err := kms.NewFilesystemKMSAndPublicKeyStorage(
		kms.FilesystemKMSConfig{
			KMSConfig: oidcKmsConfig,
			Dir:       conf.KeyDir,
			TypeID:    "oidc",
		},
	)
	if err != nil {
		return err
	}

	if err := oidcKMS.Load(); err != nil {
		return err
	}

	oidcSigner = kms.KMSToVersatileSignerWithPKStorage(
		oidcKMS,
		oidcKMS.(*kms.FilesystemKMS).PKs,
	)

	return nil
}

func getFederationSigner() jwx.VersatileSigner {
	return federationSigner
}

func getOIDCSigner() jwx.VersatileSigner {
	return oidcSigner
}
