package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
)

var (
	//go:embed static/css/output.css
	embedCSSFile []byte

	//go:embed static/htmx/htmx.js
	embedHTMXFile []byte

	//go:embed static/templates/*.html
	templatesFS embed.FS

	router *chi.Mux

	port *int

	logFilePath *string

	logFile *os.File
)

func init() {
	var err error

	port = flag.Int("port", 8080, "The port to run the application on.")
	logFilePath = flag.String("logFilePath", "", "File to write logs to. If nil, logs written to os.Stdout.")
	flag.Parse()

	if *logFilePath == "" {
		logFile = os.Stdout
	} else {
		logFile, err = os.Create(*logFilePath)
		if err != nil {
			log.Panicf("error creating httplog file: %v", err)
		}
	}

	router = chi.NewRouter()
	router.Use(httplog.RequestLogger(httplog.NewLogger("httplog", httplog.Options{
		LogLevel:       slog.LevelDebug,
		Concise:        true,
		RequestHeaders: false,
		JSON:           (*logFilePath != ""),
		Writer:         logFile,
	})))
	router.Use(middleware.Recoverer)
}

func main() {
	defer logFile.Close()

	var err error
	// applicationState := api.NewApplicationState()

	// Parse templates from embedded file system --------------------------------------------------

	templatesFS, err := fs.Sub(templatesFS, "static/templates")
	if err != nil {
		log.Fatalf("error during embedded file system: %v", err)
	}
	indexTemplate, err := template.ParseFS(templatesFS, "index.html")
	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}

	// Add handlers for CSS and HTMX files --------------------------------------------------------

	router.Get("/css/output.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Write(embedCSSFile)
	})

	router.Get("/htmx/htmx.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		w.Write(embedHTMXFile)
	})

	// Add handlers for base routes, e.g. initial page --------------------------------------------
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err = indexTemplate.Execute(w, nil)
		if err != nil {
			log.Fatalf("error during index template execute: %v", err)
		}
	})

	// Add any API routes -------------------------------------------------------------------------

	// Start server -------------------------------------------------------------------------------

	if *logFilePath == "" {
		log.Printf("Serving template at http://localhost:%v/", *port)
	}
	err = http.ListenAndServe(fmt.Sprintf(":%v", *port), router)
	if err != nil {
		log.Fatalf("error during http serving: %v", err)
	}
}
