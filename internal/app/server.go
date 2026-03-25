package app

import (
    "context"
    "net/http"
    "time"
)

type Server struct {
    httpServer *http.Server
}

func NewServer(addr string, handler http.Handler) *Server {
    return &Server{
        httpServer: &http.Server{
            Addr:              addr,
            Handler:           handler,
            ReadHeaderTimeout: 10 * time.Second,
        },
    }
}

func (s *Server) Run(ctx context.Context) error {
    errCh := make(chan error, 1)
    go func() {
        errCh <- s.httpServer.ListenAndServe()
    }()

    select {
    case <-ctx.Done():
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        _ = s.httpServer.Shutdown(shutdownCtx)
        return nil
    case err := <-errCh:
        return err
    }
}

func (s *Server) Shutdown(ctx context.Context) error {
    return s.httpServer.Shutdown(ctx)
}
