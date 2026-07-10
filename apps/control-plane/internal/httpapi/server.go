// Package httpapi holds the Control Plane HTTP server. For the bootstrap phase it
// only exposes health endpoints; real routes (auth, monitors, …) arrive later and
// will likely switch the router to Chi.
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// NewServer builds the HTTP server with health endpoints registered.
func NewServer(addr string, log *slog.Logger) *http.Server {
	return newServer(addr, log, nil)
}

// NewServerWithAuth builds the HTTP server with auth routes mounted.
func NewServerWithAuth(addr string, log *slog.Logger, authHandler http.Handler) *http.Server {
	return newServer(addr, log, authHandler)
}

func newServer(addr string, log *slog.Logger, authHandler http.Handler) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", healthHandler)
	if authHandler != nil {
		mux.Handle("/v1/auth/", authHandler)
		mux.Handle("/v1/auth", authHandler)
	}

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// Serve runs srv until ctx is cancelled, then shuts it down gracefully.
func Serve(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		log.InfoContext(ctx, "http api listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log.Info("http api shutting down")
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
