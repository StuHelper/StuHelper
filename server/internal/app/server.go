package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	gozap "go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

func (rt *Runtime) serve(router *gin.Engine) error {
	srv := &http.Server{
		Addr:              ":" + rt.cfg.App.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.L().Info("Server starting", gozap.String("port", rt.cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("failed to start server: %w", err)
		}
	}()

	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	var serverStartErr error
	select {
	case sig := <-quit:
		logger.L().Info("Received shutdown signal", gozap.String("signal", sig.String()))
		signal.Stop(quit)
	case err := <-serverErr:
		logger.L().Error("Server startup error, initiating shutdown", gozap.Error(err))
		serverStartErr = err
	}

	shutdownTimeout := time.Duration(rt.cfg.Database.QueryTimeout)*time.Second*3 + 15*time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.L().Error("Server forced to shutdown",
			gozap.Error(err),
			gozap.Duration("timeout", shutdownTimeout),
		)
	} else {
		logger.L().Info("HTTP server stopped gracefully")
	}

	if rt.bgCancel != nil {
		rt.bgCancel()
	}
	rt.runCleanups()
	logger.L().Info("All resources released, server exited")
	return serverStartErr
}
