package authserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jongio/azd-app/cli/src/internal/security"
)

// Server represents the authentication server.
type Server struct {
	config     *Config
	handler    *Handler
	httpServer *http.Server
	router     *gin.Engine
	mu         sync.Mutex
	running    bool
}

// NewServer creates a new authentication server instance.
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate paths if TLS is enabled
	if config.EnableTLS {
		if err := security.ValidatePath(config.CertFile); err != nil {
			return nil, fmt.Errorf("invalid certificate file path: %w", err)
		}
		if err := security.ValidatePath(config.KeyFile); err != nil {
			return nil, fmt.Errorf("invalid key file path: %w", err)
		}
	}

	handler, err := NewHandler(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	
	// Setup routes
	handler.SetupRoutes(router)

	return &Server{
		config:  config,
		handler: handler,
		router:  router,
	}, nil
}

// Start starts the authentication server.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:              s.config.Addr(),
		Handler:           s.router,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		var err error
		if s.config.EnableTLS {
			tlsConfig, tlsErr := s.config.TLSConfig()
			if tlsErr != nil {
				errChan <- fmt.Errorf("failed to create TLS config: %w", tlsErr)
				return
			}
			s.httpServer.TLSConfig = tlsConfig
			
			// Validate certificate and key files exist
			if _, statErr := os.Stat(s.config.CertFile); statErr != nil {
				errChan <- fmt.Errorf("certificate file not found: %w", statErr)
				return
			}
			if _, statErr := os.Stat(s.config.KeyFile); statErr != nil {
				errChan <- fmt.Errorf("key file not found: %w", statErr)
				return
			}
			
			err = s.httpServer.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
		} else {
			err = s.httpServer.ListenAndServe()
		}
		
		if err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait a bit to check if server started successfully
	select {
	case err := <-errChan:
		return fmt.Errorf("failed to start server: %w", err)
	case <-time.After(100 * time.Millisecond):
		s.running = true
		return nil
	}
}

// Stop stops the authentication server gracefully.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("server is not running")
	}

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.running = false
	return nil
}

// IsRunning returns whether the server is currently running.
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// GetURL returns the server URL.
func (s *Server) GetURL() string {
	protocol := "http"
	if s.config.EnableTLS {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s", protocol, s.config.Addr())
}
