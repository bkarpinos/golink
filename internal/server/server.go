package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"golink/internal/link"
	"golink/internal/storage"
)

// Server represents the HTTP server for go links
type Server struct {
	storage  *storage.JSONStorage
	server   *http.Server
	baseURL  string
	notFound string
}

// NewServer creates a new go links HTTP server
func NewServer(storage *storage.JSONStorage, port int, notFoundURL string) *Server {
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	return &Server{
		storage:  storage,
		baseURL:  baseURL,
		notFound: notFoundURL,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

// Start begins serving go links
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Handler for go links
	mux.HandleFunc("/", s.handleRedirect)

	// Add an information page at /info
	mux.HandleFunc("/info", s.handleInfo)

	s.server.Handler = logMiddleware(mux)

	fmt.Printf("Go Links server started at %s\n", s.baseURL)
	fmt.Printf("Press Ctrl+C to stop the server\n")

	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// handleRedirect processes go link redirects
func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	// Extract the go link alias from the path
	alias := strings.TrimPrefix(r.URL.Path, "/")

	// Empty path or root
	if alias == "" {
		s.handleRootPage(w, r)
		return
	}

	// Look up the link
	link, err := s.storage.Get(alias)
	if err != nil {
		if s.notFound != "" {
			// Redirect to the configured "not found" URL if specified
			http.Redirect(w, r, s.notFound, http.StatusFound)
			return
		}

		// If no "not found" URL is configured, show an error
		http.Error(w, fmt.Sprintf("Go link not found: %s", alias), http.StatusNotFound)
		return
	}

	// Redirect to the target URL
	http.Redirect(w, r, link.URL, http.StatusFound)
}

// handleRootPage shows a simple homepage with usage instructions
func (s *Server) handleRootPage(w http.ResponseWriter, r *http.Request) {
	links := s.storage.List()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintf(w, `<!DOCTYPE html>
	<html>
	<head>
			<title>Go Links Service</title>
			<style>
				body { font-family: monospace, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; }
        pre { white-space: pre; line-height: 1.5; }
        a { text-decoration: none; color: #0066cc; }
        a:hover { text-decoration: underline; }
			</style>
	</head>
	<body>
			<h1>Go Links Service</h1>
			<p>Use this service by navigating to <code>%s/&lt;alias&gt;</code></p>
			<h2>Available Links</h2>`, s.baseURL)

	if len(links) == 0 {
		fmt.Fprintf(w, "<p>No links available. Add some using the CLI tool.</p>")
	} else {
		// Group links by category
		categories := make(map[string][]*link.Link)
		for _, l := range links {
			cat := l.Category
			if cat == "" {
				cat = "uncategorized"
			} else {
				cat = strings.ToLower(cat) // Ensure lowercase categories
			}
			categories[cat] = append(categories[cat], l)
		}

		// Sort categories for consistent display
		sortedCats := make([]string, 0, len(categories))
		for cat := range categories {
			sortedCats = append(sortedCats, cat)
		}
		sort.Strings(sortedCats)

		// Start the pre-formatted tree output
		fmt.Fprintf(w, "<pre>")

		// Create the tree structure
		for i, cat := range sortedCats {
			catLinks := categories[cat]
			isLastCat := i == len(sortedCats)-1

			// Category prefix
			if isLastCat {
				fmt.Fprintf(w, "└── %s\n", cat)
			} else {
				fmt.Fprintf(w, "├── %s\n", cat)
			}

			// Sort links within each category
			sort.Slice(catLinks, func(i, j int) bool {
				return catLinks[i].Alias < catLinks[j].Alias
			})

			// Links under this category
			for j, l := range catLinks {
				isLastLink := j == len(catLinks)-1

				// Link prefix based on position
				if isLastCat {
					if isLastLink {
						fmt.Fprintf(w, "    └── %s → <a href=\"%s\">%s</a>\n", l.Alias, l.URL, l.URL)
					} else {
						fmt.Fprintf(w, "    ├── %s → <a href=\"%s\">%s</a>\n", l.Alias, l.URL, l.URL)
					}
				} else {
					if isLastLink {
						fmt.Fprintf(w, "│   └── %s → <a href=\"%s\">%s</a>\n", l.Alias, l.URL, l.URL)
					} else {
						fmt.Fprintf(w, "│   ├── %s → <a href=\"%s\">%s</a>\n", l.Alias, l.URL, l.URL)
					}
				}
			}
		}

		fmt.Fprintf(w, "</pre>")
	}

	fmt.Fprintf(w, `
	</body>
	</html>`)
}

// handleInfo displays information about the go links service
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	links := s.storage.List()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Go Links Service - Info</title>
    <style>
        body { font-family: monospace, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        h1, h2 { color: #333; }
        .stats { display: flex; gap: 20px; }
        .stat-box { flex: 1; padding: 15px; background: #f5f5f5; border-radius: 5px; text-align: center; }
        .stat-number { font-size: 24px; font-weight: bold; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Go Links Service - Info</h1>
    <div class="stats">
        <div class="stat-box">
            <div>Total Links</div>
            <div class="stat-number">%d</div>
        </div>
    </div>
    <h2>Service Information</h2>
    <ul>
        <li>Base URL: %s</li>
        <li>Storage: JSON File</li>
    </ul>
    <p><a href="/">Back to home</a></p>
</body>
</html>`, len(links), s.baseURL)
}

// logMiddleware logs incoming requests
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log the request
		log.Printf(
			"%s %s %s",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	})
}
