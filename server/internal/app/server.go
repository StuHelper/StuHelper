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

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
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

	// 先停止后台取件并释放 SSE 等长连接，再让 net/http 等待在途请求完成。
	// http.Server.Shutdown 本身不会主动取消活动中的长连接处理器。
	rt.beginShutdown()
	shutdownErr := shutdownHTTPServer(shutdownCtx, srv)
	if shutdownErr != nil {
		logger.L().Error("Server forced to shutdown",
			gozap.Error(shutdownErr),
			gozap.Duration("timeout", shutdownTimeout),
		)
	} else {
		logger.L().Info("HTTP server stopped gracefully")
	}

	rt.runCleanups()
	logger.L().Info("All resources released, server exited")
	return errors.Join(serverStartErr, shutdownErr)
}

type httpServerShutdown interface {
	Shutdown(context.Context) error
	Close() error
}

func shutdownHTTPServer(ctx context.Context, srv httpServerShutdown) error {
	if err := srv.Shutdown(ctx); err != nil {
		shutdownErr := fmt.Errorf("graceful HTTP shutdown: %w", err)
		if closeErr := srv.Close(); closeErr != nil {
			return errors.Join(shutdownErr, fmt.Errorf("force close HTTP server: %w", closeErr))
		}
		return shutdownErr
	}
	return nil
}
