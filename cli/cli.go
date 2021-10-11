package cli

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lab47/labctl/types"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"golang.org/x/term"
)

type CLI struct {
	c *cli.CLI
}

func NewCLI(args []string) (*CLI, error) {
	o := &CLI{}

	base := os.Getenv("LAB47_API_BASE")
	if base != "" {
		baseURL = base
	}

	o.c = cli.NewCLI("labctl", "0.1.0")
	o.c.Args = args
	o.c.Commands = map[string]cli.CommandFactory{
		"create-account": func() (cli.Command, error) {
			return newCmd(
				"create-account",
				"creates a new account on svc.lab47.dev",
				o.createF,
			), nil
		},
		"login": func() (cli.Command, error) {
			return newCmd(
				"login",
				"log into svc.lab47.dev",
				o.loginF,
			), nil
		},
		"vcr create-repo": func() (cli.Command, error) {
			return newCmd(
				"create-repo",
				"creates a new repository on vcr.pub",
				o.createRepoF,
			), nil
		},
		"vcr docker-login": func() (cli.Command, error) {
			return newCmd(
				"docker-login",
				"log the local docker instance into vcr.pub",
				o.dockerLoginF,
			), nil
		},
		"vcr kubernetes-secret": func() (cli.Command, error) {
			return newCmd(
				"kubernetes-secret",
				"print out a kubernetes secret to access vcr.pub",
				o.k8SecretF,
			), nil
		},
		"vcr namespaces": func() (cli.Command, error) {
			return newCmd(
				"namespaces",
				"list available namespaces",
				o.namespacesF,
			), nil
		},
	}

	return o, nil
}

func (c *CLI) Run() (int, error) {
	return c.c.Run()
}

func (c *CLI) loginF(ctx context.Context, opts struct {
	Email    string `short:"e" long:"email" description:"email address for account"`
	Password string `short:"p" long:"password" description:"password for account"`
}) error {
	if opts.Email == "" {
		return fmt.Errorf("email (-e) is required")
	}

	if opts.Password == "" {
		fmt.Print("Enter password: ")
		data, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return errors.Wrapf(err, "error reading password")
		}

		opts.Password = string(data)

		fmt.Println()
	}

	var tv struct {
		Token string `json:"token"`
	}

	hdrs := http.Header{}
	setAuthorization(hdrs, opts.Email, opts.Password)

	err := perform(ctx, "GET", "/api/v1/token", hdrs, nil, &tv)
	if err != nil {
		return err
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	cfg.Account.Email = opts.Email
	cfg.Account.Token = tv.Token

	err = SaveConfig(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("Logged into svc.lab47.dev!\n")

	return nil
}

func (c *CLI) createF(ctx context.Context, opts struct {
	Email     string `short:"e" long:"email" description:"email address for account"`
	Namespace string `short:"n" long:"namespace" description:"initial namespace to reserve"`
	Password  string `short:"p" long:"password" description:"password for account"`
}) error {
	if opts.Email == "" {
		return fmt.Errorf("email (-e) is required")
	}

	if opts.Namespace == "" {
		return fmt.Errorf("namespace (-n) is required")
	}

	if opts.Password == "" {
		fmt.Print("Enter password: ")
		data, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return errors.Wrapf(err, "error reading password")
		}

		opts.Password = string(data)

		fmt.Println()
	}

	fmt.Println("Creating account...")

	var tv struct {
		Token string `json:"token"`
	}

	err := Post(ctx, "/api/v1/account", map[string]string{
		"email":     opts.Email,
		"namespace": opts.Namespace,
		"password":  opts.Password,
	}, &tv)
	if err != nil {
		return err
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	cfg.Account.Email = opts.Email
	cfg.Account.Token = tv.Token

	err = SaveConfig(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("Account created and logged into!\n")

	return nil
}

func (c *CLI) createRepoF(ctx context.Context, opts struct {
	Namespace string `short:"n" long:"namespace" description:"initial namespace to reserve"`
	Pos       struct {
		Name string `positional-arg-name:"name"`
	} `positional-args:"yes"`
}) error {
	cfg, err := LoadConfig()
	if err != nil {
		return errors.Wrapf(err, "error loading configuration")
	}

	if cfg.Account.Token == "" {
		return fmt.Errorf("Please login in first")
	}

	if opts.Pos.Name == "" {
		return fmt.Errorf("requires repository name as argument")
	}

	if strings.Count(opts.Pos.Name, "/") != 1 {
		return fmt.Errorf("name must be in namespace/repo format")
	}

	fmt.Println("Creating repository...")

	fullName := opts.Pos.Name

	path := fmt.Sprintf("/vcr/v1/repo/%s", fullName)

	err = TokenPost(ctx, cfg.Account.Token, path, nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Repository created: %s\n", fullName)

	return nil
}

func (c *CLI) dockerLoginF(ctx context.Context, opts struct {
}) error {
	cfg, err := LoadConfig()
	if err != nil {
		return errors.Wrapf(err, "error loading configuration")
	}

	if cfg.Account.Token == "" {
		return fmt.Errorf("Please login first")
	}

	server := currentServer()

	fmt.Printf("Logging local docker into %s...\n", server)

	cmd := exec.CommandContext(ctx, "docker", "login", "-u", "cytoken", "-p", cfg.Account.Token, server)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (c *CLI) k8SecretF(ctx context.Context, opts struct {
}) error {
	cfg, err := LoadConfig()
	if err != nil {
		return errors.Wrapf(err, "error loading configuration")
	}

	if cfg.Account.Token == "" {
		return fmt.Errorf("Please login first")
	}

	basic := base64.StdEncoding.EncodeToString([]byte("cytoken:" + cfg.Account.Token))

	authConfg := fmt.Sprintf(`{
    "auths": {
        "https://vcr.pub": {
            "auth": "%s"
        }
    }
}
`, basic)

	encoded := base64.StdEncoding.EncodeToString([]byte(authConfg))

	str := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: vcr-pub
data:
  .dockerconfigjson: %s
type: kubernetes.io/dockerconfigjson`, encoded)

	fmt.Println(str)

	return nil
}

func (c *CLI) namespacesF(ctx context.Context, opts struct {
}) error {
	cfg, err := LoadConfig()
	if err != nil {
		return errors.Wrap(err, "error loading configuration")
	}

	if cfg.Account.Token == "" {
		return fmt.Errorf("Please login first")
	}

	var ln types.ListNamespaces

	err = TokenGet(ctx, cfg.Account.Token, "/api/v1/namespaces", &ln)
	if err != nil {
		return err
	}

	for _, ns := range ln.Namespaces {
		fmt.Printf("[namespace]\n   name: %s\ncredits: $%s\n  repos:\n", ns.Name, ns.Credit)

		for _, re := range ns.Repos {
			fmt.Printf("  - name: %s\n  - created_at: %s\n",
				re.Name,
				re.CreatedAt.Format(time.RFC3339),
			)
		}
	}

	return nil
}
