package commands

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jongio/azd-core/registry"
	"github.com/spf13/cobra"
)

const (
	defaultProxyPort       = 8080
	defaultProxyTargetHost = "localhost"
)

var proxyPort int

// NewProxyCommand creates the proxy command.
func NewProxyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Route local requests to running services",
		Long: `Start a local reverse proxy for running services.

Each running service with a local port gets a path route:
  /<service>/... -> http://localhost:<port>/...

The proxy strips the /<service> prefix before forwarding. For example,
/api/users forwards to /users on the api service.`,
		Example: `  # Start proxy on the default port
  azd app proxy

  # Start proxy on a custom port
  azd app proxy --port 9090

  # Example route table
  Proxy listening on http://localhost:8080
  /api/ -> http://localhost:5001
  /web/ -> http://localhost:3000`,
		SilenceUsage: true,
		RunE:         runProxy,
	}

	cmd.Flags().IntVar(&proxyPort, "port", defaultProxyPort, "Port for the proxy listener")

	return cmd
}

func runProxy(cmd *cobra.Command, _ []string) error {
	if proxyPort < 1 || proxyPort > 65535 {
		return fmt.Errorf("port must be between 1 and 65535: %d", proxyPort)
	}

	ctrl, err := NewServiceController("")
	if err != nil {
		return fmt.Errorf("failed to initialize service controller: %w", err)
	}

	routes, err := buildProxyRoutes(ctrl.registry.ListAll())
	if err != nil {
		return err
	}

	fmt.Printf("Proxy listening on http://localhost:%d\n", proxyPort)
	printRouteTable(routes)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", proxyPort),
		Handler:           newProxyHandler(routes),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil && !errors.Is(shutdownErr, context.Canceled) {
			return fmt.Errorf("failed to stop proxy server: %w", shutdownErr)
		}
		return nil
	case listenErr := <-errCh:
		if listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			return fmt.Errorf("failed to start proxy server on port %d: %w", proxyPort, listenErr)
		}
		return nil
	}
}

func buildProxyRoutes(entries []*registry.ServiceRegistryEntry) (map[string]*url.URL, error) {
	routes := make(map[string]*url.URL)

	for _, entry := range entries {
		if entry == nil || entry.Name == "" || entry.Port <= 0 || !isRunning(entry.Status) {
			continue
		}

		target := &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(resolveRouteHost(entry), strconv.Itoa(entry.Port)),
		}
		routes[entry.Name] = target
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("no running services with a local port were found. Start services with 'azd app run' and try again")
	}

	return routes, nil
}

func resolveRouteHost(entry *registry.ServiceRegistryEntry) string {
	if entry == nil || entry.URL == "" {
		return defaultProxyTargetHost
	}

	parsedURL, err := url.Parse(entry.URL)
	if err != nil || parsedURL.Hostname() == "" {
		return defaultProxyTargetHost
	}

	return parsedURL.Hostname()
}

func printRouteTable(routes map[string]*url.URL) {
	for _, name := range sortedRouteNames(routes) {
		fmt.Printf("/%s/ -> %s\n", name, routes[name].String())
	}
}

type proxyHandler struct {
	routes    map[string]*url.URL
	proxies   map[string]*httputil.ReverseProxy
	available string
}

func newProxyHandler(routes map[string]*url.URL) http.Handler {
	copiedRoutes := make(map[string]*url.URL, len(routes))
	proxies := make(map[string]*httputil.ReverseProxy, len(routes))

	for name, target := range routes {
		targetCopy := *target
		copiedRoutes[name] = &targetCopy
		proxies[name] = newServiceReverseProxy(name, &targetCopy)
	}

	return &proxyHandler{
		routes:    copiedRoutes,
		proxies:   proxies,
		available: strings.Join(routePrefixes(copiedRoutes), ", "),
	}
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "" || r.URL.Path == "/" {
		h.writeRouteListing(w)
		return
	}

	serviceName, rewrittenPath, ok := splitProxyPath(r.URL.Path)
	if !ok {
		h.writeNotFound(w, r.URL.Path)
		return
	}

	proxy, exists := h.proxies[serviceName]
	if !exists {
		h.writeNotFound(w, r.URL.Path)
		return
	}

	proxyRequest := cloneRequestWithPath(r, rewrittenPath)
	proxy.ServeHTTP(w, proxyRequest)
}

func (h *proxyHandler) writeRouteListing(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintln(w, "Available proxy routes:")
	for _, name := range sortedRouteNames(h.routes) {
		_, _ = fmt.Fprintf(w, "/%s/ -> %s\n", name, h.routes[name].String())
	}
}

func (h *proxyHandler) writeNotFound(w http.ResponseWriter, route string) {
	http.Error(
		w,
		fmt.Sprintf("Unknown route %q. Available routes: %s", route, h.available),
		http.StatusNotFound,
	)
}

func newServiceReverseProxy(serviceName string, target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery

	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			switch {
			case targetQuery == "":
			case req.URL.RawQuery == "":
				req.URL.RawQuery = targetQuery
			default:
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(
				w,
				fmt.Sprintf("Gateway error: service %q is unavailable: %v", serviceName, err),
				http.StatusBadGateway,
			)
		},
	}
}

func splitProxyPath(path string) (service string, rewrittenPath string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/")
	if trimmed == "" {
		return "", "", false
	}

	parts := strings.SplitN(trimmed, "/", 2)
	service = parts[0]
	if service == "" {
		return "", "", false
	}

	rewrittenPath = "/"
	if len(parts) == 2 && parts[1] != "" {
		rewrittenPath = "/" + parts[1]
	}

	return service, rewrittenPath, true
}

func cloneRequestWithPath(r *http.Request, rewrittenPath string) *http.Request {
	cloned := r.Clone(r.Context())
	clonedURL := *r.URL
	clonedURL.Path = rewrittenPath
	clonedURL.RawPath = ""
	cloned.URL = &clonedURL
	return cloned
}

func routePrefixes(routes map[string]*url.URL) []string {
	names := sortedRouteNames(routes)
	prefixes := make([]string, 0, len(names))
	for _, name := range names {
		prefixes = append(prefixes, fmt.Sprintf("/%s/", name))
	}
	return prefixes
}

func sortedRouteNames(routes map[string]*url.URL) []string {
	names := make([]string, 0, len(routes))
	for name := range routes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
