// Command bmax runs the regional services mini app for MAX.
//
//	bmax [serve]          start the HTTP server
//	bmax import <dir>     replace the catalog with <dir>/catalog.json and <dir>/media
//	bmax export <dir>     write the catalog and media into <dir>
//	bmax backup <file>    write a consistent copy of the database
//	bmax healthcheck      probe the local server (for container health checks)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgfront/bmax2/internal/config"
	"github.com/dgfront/bmax2/internal/maxauth"
	"github.com/dgfront/bmax2/internal/server"
	"github.com/dgfront/bmax2/internal/store"
	"github.com/dgfront/bmax2/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd, args := "serve", os.Args[1:]
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}

	switch cmd {
	case "serve":
		return serve(ctx, cfg, log)
	case "healthcheck":
		return healthcheck(cfg)
	case "import", "export", "backup":
		if len(args) != 1 {
			return fmt.Errorf("usage: bmax %s <path>", cmd)
		}
		st, err := store.Open(ctx, cfg.DatabasePath)
		if err != nil {
			return err
		}
		defer st.Close()
		switch cmd {
		case "import":
			err = st.ImportDir(ctx, args[0])
		case "export":
			err = st.ExportDir(ctx, args[0])
		case "backup":
			err = st.Backup(ctx, args[0])
		}
		if err == nil {
			log.Info(cmd+" done", "path", args[0])
		}
		return err
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if err := cfg.RequireServe(); err != nil {
		return err
	}
	st, err := store.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer st.Close()

	if cfg.SeedDir != "" {
		empty, err := st.IsEmpty(ctx)
		if err != nil {
			return err
		}
		if empty {
			if err := st.ImportDir(ctx, cfg.SeedDir); err != nil {
				return fmt.Errorf("seed %s: %w", cfg.SeedDir, err)
			}
			log.Info("database seeded", "dir", cfg.SeedDir)
		}
	}

	srv, err := server.New(ctx, server.Options{
		Store:      st,
		Validator:  maxauth.NewValidator(cfg.BotToken, cfg.InitDataTTL),
		AdminToken: cfg.AdminToken,
		Web:        web.Files(),
		Logger:     log,
	})
	if err != nil {
		return err
	}

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr, "admin_api", cfg.AdminToken != "")
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func healthcheck(cfg config.Config) error {
	port, err := cfg.Port()
	if err != nil {
		return err
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned %s", resp.Status)
	}
	return nil
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
