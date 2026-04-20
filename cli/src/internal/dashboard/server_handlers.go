package dashboard

import (
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/jongio/azd-core/registry"
)

// handleFallback provides a simple HTML page when the embedded dist/ static
// assets fail to load. This is an extreme failure mode (e.g., embed.FS
// initialisation error); the dashboard UI normally serves its own routes
// client-side. Only mounted by setupRoutes when fs.Sub returns an error.
func (s *Server) handleFallback(w http.ResponseWriter, r *http.Request) {
	reg := registry.GetRegistry(s.projectDir)
	services := reg.ListAll()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>AZD App Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; max-width: 1200px; margin: 40px auto; padding: 20px; }
        h1 { color: #0078d4; }
        .service { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 8px; }
    </style>
</head>
<body>
    <h1>AZD App Dashboard</h1>
    <p>Running Services in Current Project</p>
`)

	if len(services) == 0 {
		_, _ = fmt.Fprintf(w, `<p>No services are currently running.</p>`)
	} else {
		for _, svc := range services {
			escapedName := html.EscapeString(svc.Name)
			escapedURL := html.EscapeString(svc.URL)
			escapedFramework := html.EscapeString(svc.Framework)
			escapedLanguage := html.EscapeString(svc.Language)
			escapedStatus := html.EscapeString(svc.Status)

			_, _ = fmt.Fprintf(w, `
    <div class="service">
        <h3>%s</h3>
        <p><strong>URL:</strong> <a href="%s" target="_blank">%s</a></p>
        <p><strong>Framework:</strong> %s (%s)</p>
        <p><strong>Status:</strong> %s</p>
        <p><strong>Started:</strong> %s</p>
    </div>
`, escapedName, escapedURL, escapedURL, escapedFramework, escapedLanguage, escapedStatus, svc.StartTime.Format(time.RFC822))
		}
	}

	_, _ = fmt.Fprintf(w, `
</body>
</html>`)
}
