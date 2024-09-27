// Package health provides a simple health check server that responds with 200 OK to /health.
// Copyright 2024 Vega Cloud, Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package health

import (
	"context"
	"net/http"
	"time"
)

// ServerHealthCheck starts a simple web server that responds with 200 OK to /health.
// It accepts a context that can be used to cancel the server.
func ServerHealthCheck(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			return
		}
	})
	server := &http.Server{
		Addr:              ":80",
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second, // Set ReadHeaderTimeout to mitigate Slowloris attacks
	}

	// Channel to receive server errors
	errChan := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown the server with a timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	// Return any server error that occurred
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}
