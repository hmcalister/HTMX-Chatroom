package main

import (
	"bytes"
	"context"
	"embed"
	"flag"
	"fmt"
	"hmcalister/htmxChatroom/api"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
)

var (
	//go:embed static/css/output.css
	embedCSSFile []byte

	//go:embed static/htmx/htmx.js
	embedHTMXFile []byte

	//go:embed static/htmx/sse.js
	embedSSEFile []byte

	//go:embed static/templates/*.html
	templatesFS embed.FS

	router *chi.Mux

	port *int

	logFilePath *string

	logFile *os.File

	logger *httplog.Logger
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
	logger = httplog.NewLogger("httplog", httplog.Options{
		LogLevel:       slog.LevelDebug,
		Concise:        true,
		RequestHeaders: false,
		JSON:           (*logFilePath != ""),
		Writer:         logFile,
	})
	router.Use(httplog.RequestLogger(logger))
	router.Use(middleware.Recoverer)
}

func main() {
	defer logFile.Close()

	var err error
	applicationState := api.NewApplicationState()

	// Parse templates from embedded file system --------------------------------------------------

	templatesFS, err := fs.Sub(templatesFS, "static/templates")
	if err != nil {
		logger.Error(fmt.Sprintf("error during embedded file system: %v", err))
	}
	indexTemplate, err := template.ParseFS(templatesFS, "index.html")
	if err != nil {
		logger.Error(fmt.Sprintf("error parsing template: %v", err))
	}
	messageTemplate, err := template.ParseFS(templatesFS, "message.html")
	if err != nil {
		logger.Error(fmt.Sprintf("error parsing template: %v", err))
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

	router.Get("/htmx/sse.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		w.Write(embedSSEFile)
	})

	// Add handlers for base routes, e.g. initial page --------------------------------------------
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err = indexTemplate.Execute(w, nil)
		if err != nil {
			logger.Error(fmt.Sprintf("error during index template execute: %v", err))
		}
	})

	// Add any API routes -------------------------------------------------------------------------

	router.Get("/api/chatroomSSE", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		messageChannel := make(chan string)
		_, cancel := context.WithCancel(r.Context())
		defer cancel()

		go func() {
			for message := range messageChannel {
				for _, line := range strings.Split(strings.TrimSuffix(message, "\n"), "\n") {
					fmt.Fprintf(w, "data: %s\n", line)
				}
				fmt.Fprint(w, "\n\n")
				w.(http.Flusher).Flush()
			}
		}()

		for {
			var messageTemplateBuffer bytes.Buffer
			var messageString string
			select {
			case <-r.Context().Done():
				logger.Warn(fmt.Sprintf("err context canceled: %v", err))
				return
			default:
				err = messageTemplate.Execute(&messageTemplateBuffer, api.MessageTemplateData{
					SenderName: "DEFAULT",
					Message:    fmt.Sprintf("Message %v", applicationState.NewMessage()),
				})
				if err != nil {
					logger.Error(fmt.Sprintf("err during message execute: %v", err))
					return
				}
				messageString = messageTemplateBuffer.String()
				messageChannel <- messageString
				time.Sleep(1 * time.Second)
			}
		}
	})

	// Start server -------------------------------------------------------------------------------

	if *logFilePath == "" {
		logger.Info(fmt.Sprintf("Serving template at http://localhost:%v/", *port))
	}
	err = http.ListenAndServe(fmt.Sprintf(":%v", *port), router)
	if err != nil {
		logger.Error(fmt.Sprintf("error during http serving: %v", err))
	}
}
