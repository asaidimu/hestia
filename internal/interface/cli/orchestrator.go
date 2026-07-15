package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/internal/abstract"
	"github.com/asaidimu/hestia/internal/core"
)

var systemIdentity = iam.Identity{
	Permissions: []string{core.SystemScopePrefix + ":cli"},
	Properties:  map[string]any{"system": "cli"},
}

type Options struct {
	Dispatcher  core.Dispatcher
	Logger      *zap.Logger
	AdminUserID string
	AdminEmail  string
	Version     string
	Stdin       io.Reader
	Stdout      io.Writer
}

type Orchestrator struct {
	opts Options
}

func New(opts Options) *Orchestrator {
	return &Orchestrator{opts: opts}
}

func (o *Orchestrator) Start(bootstrapped bool) {
	if len(os.Args) < 2 {
		return
	}

	fs := flag.NewFlagSet("hestia", flag.ContinueOnError)
	fs.SetOutput(o.opts.Stdout)

	version := fs.Bool("v", false, "print version")
	versionAlias := fs.Bool("version", false, "print version")
	help := fs.Bool("h", false, "print help")
	helpAlias := fs.Bool("help", false, "print help")
	bootstrap := fs.Bool("bootstrap", false, "run interactive bootstrap (set admin password)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	switch {
	case *version || *versionAlias:
		o.printVersion()
		os.Exit(0)
	case *help || *helpAlias:
		o.printHelp(fs)
		os.Exit(0)
	case *bootstrap:
		o.runBootstrap()
		os.Exit(0)
	}
}

func (o *Orchestrator) Restart(bootstrapped bool) {}

func (o *Orchestrator) Shutdown(ctx context.Context) error { return nil }

func (o *Orchestrator) printVersion() {
	fmt.Fprintln(o.opts.Stdout, "Hestia ERP Template Server", o.opts.Version)
}

func (o *Orchestrator) printHelp(fs *flag.FlagSet) {
	fmt.Fprintln(o.opts.Stdout, "Hestia ERP Template Server")
	fmt.Fprintln(o.opts.Stdout)
	fmt.Fprintln(o.opts.Stdout, "Usage:")
	fmt.Fprintln(o.opts.Stdout, "  hestia [flags]")
	fmt.Fprintln(o.opts.Stdout)
	fmt.Fprintln(o.opts.Stdout, "Flags:")
	fs.PrintDefaults()
	fmt.Fprintln(o.opts.Stdout)
	fmt.Fprintln(o.opts.Stdout, "Without flags, the server starts in HTTP mode.")
}

func (o *Orchestrator) runBootstrap() {
	ctx := iam.WithIdentity(context.Background(), systemIdentity)

	statusMsg := abstract.NewMessage("system:core:health:check", ctx, data.MustNewDocument(nil, ctx))
	statusResult, err := o.opts.Dispatcher.Send(statusMsg)
	if err != nil {
		fmt.Fprintf(o.opts.Stdout, "Failed to check system status: %v\n", err)
		os.Exit(1)
	}
	bootstrapped := false
	if statusResult != nil && statusResult.Document != nil {
		if v, _ := statusResult.Document.GetOr("bootstrapped", false).(bool); v {
			bootstrapped = true
		}
	}
	if bootstrapped {
		fmt.Fprintln(o.opts.Stdout, "System is already bootstrapped.")
		return
	}

	scanner := bufio.NewScanner(o.opts.Stdin)

	fmt.Fprint(o.opts.Stdout, "Admin email: ")
	if !scanner.Scan() {
		os.Exit(1)
	}
	email := strings.TrimSpace(scanner.Text())
	if email == "" {
		fmt.Fprintln(o.opts.Stdout, "Email is required.")
		os.Exit(1)
	}

	fmt.Fprint(o.opts.Stdout, "New password: ")
	if !scanner.Scan() {
		os.Exit(1)
	}
	password := scanner.Text()
	if password == "" {
		fmt.Fprintln(o.opts.Stdout, "Password is required.")
		os.Exit(1)
	}

	fmt.Fprint(o.opts.Stdout, "Confirm password: ")
	if !scanner.Scan() {
		os.Exit(1)
	}
	confirm := scanner.Text()
	if password != confirm {
		fmt.Fprintln(o.opts.Stdout, "Passwords do not match.")
		os.Exit(1)
	}

	pwdMsg := abstract.NewMessage("system:auth:bootstrap:password:set", ctx, data.MustNewDocument(map[string]any{
		"payload": map[string]any{"password": password, "email": email, "caller_id": o.opts.AdminUserID},
	}, ctx))
	if _, err := o.opts.Dispatcher.Send(pwdMsg); err != nil {
		fmt.Fprintf(o.opts.Stdout, "Bootstrap failed: %v\n", err)
		os.Exit(1)
	}

	bsMsg := abstract.NewMessage("system:core:bootstrap:mark", ctx, data.MustNewDocument(nil, ctx))
	if _, err := o.opts.Dispatcher.Send(bsMsg); err != nil {
		fmt.Fprintf(o.opts.Stdout, "Failed to mark bootstrapped: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(o.opts.Stdout, "Bootstrap complete. Start the server normally to use all features.")
}


