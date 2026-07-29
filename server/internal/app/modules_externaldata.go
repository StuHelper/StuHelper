package app

import (
	"fmt"
	"strings"
	"time"

	gozap "go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/externaldata"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

func (rt *Runtime) initExternalStudentDirectory() (*externaldata.StudentDirectoryRegistry, error) {
	if len(rt.cfg.ExternalData.StudentSources) == 0 {
		return nil, nil
	}
	sources := make([]externaldata.StudentSource, 0, len(rt.cfg.ExternalData.StudentSources))
	for _, source := range rt.cfg.ExternalData.StudentSources {
		if !source.Enabled {
			continue
		}
		directory, err := rt.buildExternalStudentDirectory(source)
		if err != nil {
			return nil, err
		}
		sources = append(sources, externaldata.StudentSource{
			Name:       source.Name,
			SchoolCode: source.SchoolCode,
			Directory:  directory,
		})
	}
	if len(sources) == 0 {
		return nil, nil
	}
	registry, err := externaldata.NewStudentDirectoryRegistry(sources)
	if err != nil {
		return nil, fmt.Errorf("initialize external student directory registry: %w", err)
	}
	rt.addCleanup(func() {
		if closeErr := registry.Close(); closeErr != nil {
			logger.L().Warn("external student directory registry close error", gozap.Error(closeErr))
		}
	})
	return registry, nil
}

func (rt *Runtime) buildExternalStudentDirectory(
	source config.ExternalStudentSourceConfig,
) (externaldata.StudentDirectory, error) {
	switch strings.TrimSpace(source.Provider) {
	case "oracle":
		return externaldata.NewOracleStudentDirectory(externaldata.OracleStudentDirectoryConfig{
			SchoolCode:              source.SchoolCode,
			Host:                    source.Oracle.Host,
			Port:                    source.Oracle.Port,
			ServiceName:             source.Oracle.ServiceName,
			Username:                source.Oracle.Username,
			Password:                source.Oracle.Password,
			TLSMode:                 source.Oracle.TLSMode,
			TLSCAFile:               source.Oracle.TLSCAFile,
			Schema:                  source.Oracle.Schema,
			Table:                   source.Oracle.Table,
			StudentIDColumn:         source.Oracle.StudentIDColumn,
			StudentNameColumn:       source.Oracle.StudentNameColumn,
			ConnectTimeout:          time.Duration(source.Oracle.ConnectTimeoutSeconds) * time.Second,
			QueryTimeout:            time.Duration(source.Oracle.QueryTimeoutSeconds) * time.Second,
			MaxOpenConns:            source.Oracle.MaxOpenConns,
			MaxIdleConns:            source.Oracle.MaxIdleConns,
			ConnMaxLifetime:         time.Duration(source.Oracle.ConnMaxLifetimeSeconds) * time.Second,
			ConnMaxIdleTime:         time.Duration(source.Oracle.ConnMaxIdleTimeSeconds) * time.Second,
			BreakerFailureThreshold: source.Oracle.BreakerFailureThreshold,
			BreakerSuccessThreshold: source.Oracle.BreakerSuccessThreshold,
			BreakerOpenTimeout:      time.Duration(source.Oracle.BreakerOpenSeconds) * time.Second,
		})
	default:
		return nil, fmt.Errorf("unsupported external student source provider %q", source.Provider)
	}
}
