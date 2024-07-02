package caddy_anycable

import (
	"context"
	"fmt"
	"github.com/anycable/anycable-go/cli"
	"github.com/anycable/anycable-go/config"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var caddyLogger = NewCaddyLogHandler()

func init() {
	caddy.RegisterModule(AnyCableHandler{})
	httpcaddyfile.RegisterHandlerDirective("anycable", parseCaddyfile)
}

type AnyCableHandler struct {
	logger     *slog.Logger
	config     *config.Config
	anycable   *cli.Embedded
	wsHandler  http.Handler
	sseHandler http.Handler
	Options    []string
}

func (AnyCableHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.anycable",
		New: func() caddy.Module { return new(AnyCableHandler) },
	}
}

func (h AnyCableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if h.config.SSE.Enabled {
		if matchPath(h.config.SSE.Path, r.URL.Path) {
			h.sseHandler.ServeHTTP(w, r)
			return nil
		}
	}

	for _, path := range h.config.Path {
		if matchPath(path, r.URL.Path) {
			h.wsHandler.ServeHTTP(w, r)
			return nil
		}
	}

	return next.ServeHTTP(w, r)
}

func (h *AnyCableHandler) Cleanup() error {
	if h.anycable != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.config.App.ShutdownTimeout)*time.Second)
		defer cancel()

		err := h.anycable.Shutdown(ctx)
		if err != nil {
			h.logger.Error("Error shutting down AnyCable: ", err)
			return err
		}
	}

	return nil
}

func (h AnyCableHandler) initConfig() (*config.Config, error) {
	argsWithProg := append([]string{"anycable-go"}, h.Options...)
	c, err, _ := cli.NewConfigFromCLI(argsWithProg)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (h AnyCableHandler) runAnyCable() (*cli.Embedded, error) {
	opts := []cli.Option{
		cli.WithName("AnyCable"),
		cli.WithDefaultRPCController(),
		cli.WithDefaultBroker(),
		cli.WithDefaultSubscriber(),
		cli.WithDefaultBroadcaster(),
		cli.WithLogger(h.logger),
	}

	runner, err := cli.NewRunner(h.config, opts)
	if err != nil {
		return nil, err
	}

	anycable, err := runner.Embed()
	return anycable, err
}

func (h *AnyCableHandler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			key := d.Val()
			if d.NextArg() {
				h.Options = append(h.Options, fmt.Sprintf("--%s=%s", key, d.Val()))

				if d.NextArg() {
					return d.Errf("expected only 1 argument for %s", key)
				}
			} else {
				return d.Errf("expected 1 argument for %s but none provided", key)
			}
		}
	}

	return nil
}

func (h *AnyCableHandler) Provision(ctx caddy.Context) error {
	logger := slog.New(caddyLogger)
	h.logger = logger

	cfg, err := h.initConfig()
	if err != nil {
		return err
	}

	h.config = cfg

	anycable, err := h.runAnyCable()
	if err != nil {
		return err
	}

	h.anycable = anycable
	h.wsHandler, _ = anycable.WebSocketHandler()
	h.sseHandler, _ = anycable.SSEHandler(ctx)

	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var anyCable AnyCableHandler
	err := anyCable.UnmarshalCaddyfile(h.Dispenser)
	return anyCable, err
}

func matchPath(pattern, path string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	return pattern == path
}

var (
	_ caddyhttp.MiddlewareHandler = (*AnyCableHandler)(nil)
	_ caddy.Provisioner           = (*AnyCableHandler)(nil)
	_ caddyfile.Unmarshaler       = (*AnyCableHandler)(nil)
)
