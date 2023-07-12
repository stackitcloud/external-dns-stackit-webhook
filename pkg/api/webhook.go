package api

import (
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/provider"
)

type webhook struct {
	provider provider.Provider
	logger   *zap.Logger
}
