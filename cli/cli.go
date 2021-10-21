package cli

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lab47/labctl/types"
	"github.com/mitchellh/cli"
	"github.com/pkg/browser"
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
		"namespaces": func() (cli.Command, error) {
			return newCmd(
				"namespaces",
				"list available namespaces",
				o.namespacesF,
			), nil
		},
		"machine-account create": func() (cli.Command, error) {
			return newCmd(
				"machine-account-create",
				"create a new machine account",
				o.createMachineF,
			), nil

		},
		"credit add": func() (cli.Command, error) {
			return newCmd(
				"created-add",
				"add credit to a namespace",
				o.creditAddF,
			), nil
		},
		"vcr create-repo": func() (cli.Command, error) {
			return newCmd(
				"create-repo",
				"creates a new repository on vcr.pub",
				o.createRepoF,
			), nil
		},
		"vcr update-repo": func() (cli.Command, error) {
			return newCmd(
				"update-repo",
				"update repository settings",
				o.repoSettingsF,
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
		"vcr util read-manifest": func() (cli.Command, error) {
			return newCmd(
				"read-manifest",
				"print out the manifest for a given reference",
				o.fetchManifestF,
			), nil
		},
		"vcr util read-config": func() (cli.Command, error) {
			return newCmd(
				"read-config",
				"print out the config for a given reference",
				o.fetchConfigF,
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

func (c *CLI) createMachineF(ctx context.Context, opts struct {
	Namespace   string `short:"n" long:"namespace" description:"initial namespace to reserve"`
	Name        string `long:"name" description:"name for machine account"`
	Description string `short:"d" long:"description" description:"description of machine account"`
	Write       bool   `long:"enable-write" description:"allow the account to have write access"`
}) error {
	if opts.Namespace == "" {
		return fmt.Errorf("namespace (-n) is required")
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	name := opts.Name
	if name == "" {
		name = fmt.Sprintf("machine-%s", uuid.New().String())
	}

	fmt.Printf("Creating machine account '%s'...\n", name)

	var tv types.MachineAccountCreateResponse

	path := fmt.Sprintf("/api/v1/namespace/%s/machine-account", opts.Namespace)

	err = TokenPut(ctx, cfg.Account.Token, path, &types.MachineAccountCreateRequest{
		Name:        name,
		Description: opts.Description,
		Write:       opts.Write,
	}, &tv)
	if err != nil {
		return err
	}

	fmt.Printf("Machine account created!\nToken for account: %s\n", tv.Token)

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

func (c *CLI) repoSettingsF(ctx context.Context, opts struct {
	Public  *bool `short:"P" long:"public" description:"change the repos to public"`
	Private *bool `short:"R" long:"private" description:"change the repos to private"`

	Pos struct {
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

	fmt.Println("Updating repository settings...")

	fullName := opts.Pos.Name

	path := fmt.Sprintf("/vcr/v1/repo/%s/update-settings", fullName)

	var settings types.RepoSettingsApply

	if opts.Private != nil {
		if opts.Public != nil {
			return errors.New("Set either -P or -R, not both")
		}

		pub := false

		settings.Public = &pub
		fmt.Printf("=> Setting visibility to private\n")
	} else if opts.Public != nil {
		settings.Public = opts.Public
		fmt.Printf("=> Setting visibility to public\n")
	}

	err = TokenPut(ctx, cfg.Account.Token, path, settings, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Updated %s!\n", fullName)

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

const winClose = `<html>
<body>
<script>
	window.close()
</script>
	<h4>
	You may now close this window.
	</h4>
</body>
</html>
`

func (c *CLI) creditAddF(ctx context.Context, opts struct {
	Namespace string `short:"n" long:"namespace" description:"initial namespace to reserve"`
	Dollars   int64  `short:"d" long:"credit" description:"how many USD to add in credits"`
}) error {
	cfg, err := LoadConfig()
	if err != nil {
		return errors.Wrapf(err, "error loading configuration")
	}

	if cfg.Account.Token == "" {
		return fmt.Errorf("Please login in first")
	}

	if opts.Namespace == "" {
		return fmt.Errorf("name of namespace required")
	}

	if opts.Dollars == 0 {
		return fmt.Errorf("number of US Dollars to add to namespace required")
	}

	fmt.Printf("Requesting $%d USD to namespace %s...\n", opts.Dollars, opts.Namespace)

	path := "/api/v1/credit/add"

	req := &types.CreditAddRequest{
		Namespace: opts.Namespace,
		Credits:   opts.Dollars,
	}

	l, err := net.Listen("tcp", ":0")
	if err == nil {
		req.LocalPort = l.Addr().(*net.TCPAddr).Port
	}

	var resp types.CreditAddResponse

	err = TokenPut(ctx, cfg.Account.Token, path, req, &resp)
	if err != nil {
		return err
	}

	fmt.Println("Opening browser to enter payment information!")

	err = browser.OpenURL(resp.URL)
	if err != nil {
		fmt.Printf("Error opening browser. Please go to:\n%s\n", resp.URL)
		return nil
	}

	if l == nil {
		fmt.Println("Use payment screen to complete payment and credits will be added to account.")
		return nil
	}

	var (
		status  string
		balance string
	)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	h := &http.Server{
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			status = r.FormValue("status")
			balance = r.FormValue("balance")

			fmt.Fprintf(rw, winClose)
			cancel()
		}),
	}

	defer h.Shutdown(context.Background())

	fmt.Println("Waiting for payment to complete...")
	go h.Serve(l)

	<-ctx.Done()

	switch status {
	case "":
		fmt.Println("Timed out waiting for signal of successful payment.")
		fmt.Println("Credits may by added anyway, check `labctl namespaces`.")
	case "success":
		fmt.Printf("Credits added! Current balance: %s\n", balance)
	case "cancel":
		fmt.Println("Payment canceled, no credits added.")
	}

	return nil
}
