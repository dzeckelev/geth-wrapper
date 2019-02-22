package api

import (
	"net/http"

	"github.com/dzeckelev/geth-wrapper/config"

	"github.com/ethereum/go-ethereum/rpc"
)

// Server is a RPC server.
type Server struct {
	rpcSrv  *rpc.Server
	httpSrv *http.Server
}

// NewServer creates a new API server.
func NewServer(cfg *config.Config) (*Server, error) {
	rpcSrv := rpc.NewServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/", rpcSrv.ServeHTTP)

	httpSrv := &http.Server{
		Addr:    cfg.API.Addr,
		Handler: mux,
	}

	return &Server{
		rpcSrv:  rpcSrv,
		httpSrv: httpSrv,
	}, nil
}

// AddHandler registers a new RPC handler.
func (s *Server) AddHandler(handler interface{}) error {
	return s.rpcSrv.RegisterName("api", handler)
}

// ListenAndServe starts to listen and to serve requests.
func (s *Server) ListenAndServe() error {
	// TODO: Enable TLS
	return s.httpSrv.ListenAndServe()
}

// Close closes the server.
func (s *Server) Close() error {
	return s.httpSrv.Close()
}
