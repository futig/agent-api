package common

import (
	"github.com/futig/agent-backend/internal/config"
	pkgHTTP "github.com/futig/agent-backend/pkg/http"
	"go.uber.org/zap"
)

func NewBaseConnector(cfg config.HTTPClientConfig, logger *zap.Logger) *pkgHTTP.Connector {
	connCfg := &pkgHTTP.ConnectorConfig{
		Logger:  logger,
		BaseURL: cfg.Url,
	}

	return pkgHTTP.NewConnector(
		connCfg,
		pkgHTTP.WithRequestTimeout(cfg.RequestTimeout),
		pkgHTTP.WithConnClientTimeout(cfg.ConnTimeout),
		pkgHTTP.WithClientKeepAlive(cfg.KeepAlive),
		pkgHTTP.WithIdleConnTimeout(cfg.IdleConnTimeout),
		pkgHTTP.WithResponseHeaderTimeout(cfg.ResponseHeaderTimeout),
		pkgHTTP.WithRequestLogging(),
		pkgHTTP.WithAuthToken(cfg.Token),
	)
}
