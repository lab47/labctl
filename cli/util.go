package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/lab47/labctl/pkg/fulcioroots"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/oci"
	coremote "github.com/sigstore/cosign/pkg/oci/remote"
	sigs "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
)

func (c *CLI) fetchSigF(ctx context.Context, opts struct {
	Username string `short:"u" description:"username to authenticate with"`
	Password string `short:"p" description:"password associated with username"`

	Pos struct {
		Name string `positional-arg-name:"name" required:"true"`
	} `positional-args:"yes" required:"true"`
}) error {
	ref, err := name.ParseReference(opts.Pos.Name)
	if err != nil {
		return errors.Wrapf(err, "error parse reference")
	}

	var ropts []remote.Option

	if opts.Password != "" {
		ropts = append(ropts, remote.WithAuth(&authn.Basic{
			Username: opts.Username,
			Password: opts.Password,
		}))
	}

	var co cosign.CheckOpts
	co.RekorURL = "https://rektor.sigstore.dev"
	co.RootCerts = fulcioroots.Get()
	co.ClaimVerifier = cosign.SimpleClaimVerifier

	co.RegistryClientOpts = append(co.RegistryClientOpts, coremote.WithRemoteOptions(ropts...))

	sigs, bundled, err := cosign.VerifySignatures(ctx, ref, &co)
	if err != nil {
		return err
	}

	PrintVerificationHeader(opts.Pos.Name, &co, bundled)
	PrintVerification(opts.Pos.Name, sigs)

	return nil
}

func PrintVerificationHeader(imgRef string, co *cosign.CheckOpts, bundleVerified bool) {
	fmt.Fprintln(os.Stderr, "Checks:")
	fmt.Fprintln(os.Stderr, "✅ fulcio roots")
	if co.ClaimVerifier != nil {
		fmt.Fprintln(os.Stderr, "✅ cosign claims")
	}
	if bundleVerified {
		fmt.Fprintln(os.Stderr, "✅ offline tranparency log")
	} else if co.RekorURL != "" {
		fmt.Fprintln(os.Stderr, "✅ online transparency log")
	}
}

// PrintVerification logs details about the verification to stdout
func PrintVerification(imgRef string, verified []oci.Signature) {
	for _, sig := range verified {
		if cert, err := sig.Cert(); err == nil && cert != nil {
			fmt.Println("✅ subject:", sigs.CertSubject(cert))
			if issuerURL := sigs.CertIssuerExtension(cert); issuerURL != "" {
				fmt.Println("✅ issuer:", issuerURL)
			}
		}

		p, err := sig.Payload()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching payload: %v", err)
			return
		}

		ss := payload.SimpleContainerImage{}
		if err := json.Unmarshal(p, &ss); err != nil {
			fmt.Println("error decoding the payload:", err.Error())
			return
		}

		if ss.Optional["signed-by"] == "vcr.pub" {
			fmt.Fprintln(os.Stderr, "✅ vcr.pub service side signature")
		}
	}
}

func (c *CLI) fetchManifestF(ctx context.Context, opts struct {
	Username string `short:"u" description:"username to authenticate with"`
	Password string `short:"p" description:"password associated with username"`

	Pos struct {
		Name string `positional-arg-name:"name" required:"true"`
	} `positional-args:"yes" required:"true"`
}) error {

	ref, err := name.ParseReference(opts.Pos.Name)
	if err != nil {
		return errors.Wrapf(err, "error parse reference")
	}

	var ropts []remote.Option

	if opts.Password != "" {
		ropts = append(ropts, remote.WithAuth(&authn.Basic{
			Username: opts.Username,
			Password: opts.Password,
		}))
	}

	desc, err := remote.Get(ref, ropts...)
	if err != nil {
		return errors.Wrapf(err, "error reading manifest")
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	fmt.Println("Descriptor:")
	enc.Encode(map[string]interface{}{
		"annotations": desc.Annotations,
		"digest":      desc.Digest,
		"media-type":  desc.MediaType,
	})

	fmt.Println("Manifest:")
	switch desc.MediaType {
	case "application/vnd.docker.distribution.manifest.list.v2+json", v1.MediaTypeImageIndex:
		var man v1.Index

		err = json.Unmarshal(desc.Manifest, &man)
		if err != nil {
			return errors.Wrapf(err, "error parsing manifest")
		}
		return enc.Encode(&man)
	case "application/vnd.docker.distribution.manifest.v2+json", v1.MediaTypeImageManifest:

		var man v1.Manifest

		err = json.Unmarshal(desc.Manifest, &man)
		if err != nil {
			return errors.Wrapf(err, "error parsing manifest")
		}
		return enc.Encode(&man)
	default:
		return errors.Errorf("unknown media-type: %s", desc.MediaType)
	}
}

func (c *CLI) fetchConfigF(ctx context.Context, opts struct {
	Username string `short:"u" description:"username to authenticate with"`
	Password string `short:"p" description:"password associated with username"`

	Pos struct {
		Name string `positional-arg-name:"name" required:"true"`
	} `positional-args:"yes" required:"true"`
}) error {
	ref, err := name.ParseReference(opts.Pos.Name)
	if err != nil {
		return errors.Wrapf(err, "error parse reference")
	}

	desc, err := remote.Get(ref, remote.WithAuth(&authn.Basic{
		Username: opts.Username,
		Password: opts.Password,
	}))
	if err != nil {
		return errors.Wrapf(err, "error reading manifest")
	}

	img, err := desc.Image()
	if err != nil {
		return errors.Wrapf(err, "error parsing image information")
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return errors.Wrapf(err, "error parsing config information")
	}

	err = json.Unmarshal(desc.Manifest, cfg)
	if err != nil {
		return errors.Wrapf(err, "error parsing manifest")
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(cfg)
}
