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

	gozap "go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto/pii"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

// runCampusConnectorBootstrap starts only the mutually authenticated campus
// gateway and the roster importer required for a first production snapshot.
// It deliberately does not build the public API router or initialize OIDC,
// OpenFGA, token services, or application background workers.
func (rt *Runtime) runCampusConnectorBootstrap() error {
	if err := rt.initSharedRuntimeServices(); err != nil {
		rt.runCleanups()
		return err
	}

	piiCipher, err := pii.NewCipher(
		rt.cfg.Security.DocAESActiveKeyID,
		rt.cfg.Security.DocAESKeys,
	)
	if err != nil {
		rt.runCleanups()
		return fmt.Errorf("failed to initialize roster PII cipher: %w", err)
	}

	connectorService, err := rt.initCampusConnectorGateway()
	if err != nil {
		rt.runCleanups()
		return err
	}
	if connectorService == nil {
		rt.runCleanups()
		return errors.New("campus connector gateway is disabled in bootstrap mode")
	}

	studentVerificationService, err := studentverification.NewService(
		studentverification.NewRepository(rt.database),
		crypto.GetHMACKey(),
		studentverification.WithRosterCipher(
			piiCipher,
			int(rt.cfg.Security.DocAESActiveKeyID),
		),
	)
	if err != nil {
		rt.runCleanups()
		return fmt.Errorf("initialize campus connector roster importer: %w", err)
	}
	connectorService.SetSnapshotImporter(
		campusConnectorSnapshotImporter{service: studentVerificationService},
	)

	return rt.serveCampusConnectorBootstrap()
}

func (rt *Runtime) serveCampusConnectorBootstrap() error {
	listener, err := rt.campusConnector.listen()
	if err != nil {
		rt.runCleanups()
		return fmt.Errorf("failed to start campus connector bootstrap listener: %w", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.L().Info(
			"Campus connector bootstrap gateway starting",
			gozap.String("address", rt.campusConnector.address),
		)
		if serveErr := rt.campusConnector.server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("campus connector bootstrap gateway failed: %w", serveErr)
		}
	}()

	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	var serveErr error
	select {
	case sig := <-quit:
		logger.L().Info(
			"Campus connector bootstrap received shutdown signal",
			gozap.String("signal", sig.String()),
		)
	case serveErr = <-serverErr:
		logger.L().Error(
			"Campus connector bootstrap gateway error, initiating shutdown",
			gozap.Error(serveErr),
		)
	}

	shutdownTimeout := time.Duration(rt.cfg.Database.QueryTimeout)*time.Second*3 + 15*time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	shutdownErr := shutdownHTTPServer(shutdownCtx, rt.campusConnector.server)
	if shutdownErr != nil {
		shutdownErr = fmt.Errorf("campus connector bootstrap gateway shutdown: %w", shutdownErr)
	}
	rt.runCleanups()
	return errors.Join(serveErr, shutdownErr)
}
