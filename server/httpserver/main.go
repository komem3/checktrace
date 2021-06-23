package main

import (
	"net/http"
	"os"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/go-chi/chi/v5"
	"github.com/komem3/checktrace/internal/gcp"
	"github.com/komem3/checktrace/internal/logger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

func main() {
	log := logger.NewLogger()

	project, err := gcp.Project("test")
	if err != nil {
		log.Panic().Err(err).Msg("")
	}
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: project,
	})
	if err != nil {
		log.Panic().Err(err).Msg("")
	}
	trace.RegisterExporter(exporter)

	router := chi.NewRouter()
	router.Use(logger.LogMiddleware(project, log))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := trace.StartSpan(r.Context(), "httpget")
		defer span.End()
		l := logger.LoggerFromContext(ctx)
		values := r.URL.Query()
		value := values.Get("value")

		l.Info().Str("value", value).Msg("")
		if value == "error" {
			l.Err(err).Msg("")
		}

		w.Write([]byte("value=" + value))
	})

	httpHandler := &ochttp.Handler{
		Handler: router,
		// Use the Google Cloud propagation format.
		Propagation: &propagation.HTTPFormat{},
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Debug().Str("port", port).Msg("bind port")
	log.Panic().Err(http.ListenAndServe(":"+port, httpHandler)).Msg("")
}
