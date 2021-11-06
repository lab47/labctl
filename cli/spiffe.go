package cli

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/davecgh/go-spew/spew"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/lab47/labctl/types"
	"github.com/sigstore/fulcio/pkg/client"
	"github.com/sigstore/fulcio/pkg/generated/client/operations"
	"github.com/sigstore/fulcio/pkg/generated/models"
	"gopkg.in/square/go-jose.v2/jwt"
)

func (c *CLI) personalToken(ctx context.Context, opts struct {
	Validate bool `short:"V" long:"validate" description:"validate token for OIDC"`
}) error {
	var req types.PersonalTokenRequest

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	var ret types.PersonalTokenResponse

	err = TokenPost(ctx, cfg.Account.Token, "https://allow.pub/api/v1/personal-token", &req, &ret)
	if err != nil {
		return err
	}

	fmt.Println(ret.JWT)

	if opts.Validate {
		prov, err := oidc.NewProvider(context.Background(), "https://allow.pub")
		if err != nil {
			return err
		}
		ver := prov.Verifier(&oidc.Config{
			SkipClientIDCheck: true,
		})

		ot, err := ver.Verify(ctx, ret.JWT)
		if err != nil {
			return err
		}

		spew.Dump(ot)
	}
	return nil
}

func (c *CLI) fulcioCert(ctx context.Context, opts struct {
	Validate bool `short:"V" long:"validate" description:"validate token for OIDC"`
}) error {
	var req types.PersonalTokenRequest

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	var ret types.PersonalTokenResponse

	err = TokenPost(ctx, cfg.Account.Token, "https://allow.pub/api/v1/personal-token", &req, &ret)
	if err != nil {
		return err
	}

	u, err := url.Parse(client.SigstorePublicServerURL)
	if err != nil {
		return err
	}

	fc := client.New(u)

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return err
	}

	tok, err := jwt.ParseSigned(ret.JWT)
	if err != nil {
		return err
	}

	var claims jwt.Claims

	err = tok.UnsafeClaimsWithoutVerification(&claims)
	if err != nil {
		return err
	}

	// Sign the email address as part of the request
	h := sha256.Sum256([]byte(claims.Subject))
	proof, err := ecdsa.SignASN1(rand.Reader, priv, h[:])
	if err != nil {
		return err
	}

	bearerAuth := httptransport.BearerToken(ret.JWT)

	content := strfmt.Base64(pubBytes)
	signedChallenge := strfmt.Base64(proof)
	params := operations.NewSigningCertParams()
	params.SetCertificateRequest(
		&models.CertificateRequest{
			PublicKey: &models.CertificateRequestPublicKey{
				Algorithm: models.CertificateRequestPublicKeyAlgorithmEcdsa,
				Content:   &content,
			},
			SignedEmailAddress: &signedChallenge,
		},
	)

	resp, err := fc.Operations.SigningCert(params, bearerAuth)
	if err != nil {
		return err
	}

	// split the cert and the chain
	// certBlock, chainPem := pem.Decode([]byte(resp.Payload))
	// certPem := pem.EncodeToMemory(certBlock)

	fmt.Println(resp.Payload)
	fmt.Println(resp.SCT.String())

	return nil
}
