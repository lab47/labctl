package cli

import (
	"context"
	"encoding/json"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

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

	enc.Encode(map[string]interface{}{
		"annotations": desc.Annotations,
		"digest":      desc.Digest,
		"media-type":  desc.MediaType,
	})

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
