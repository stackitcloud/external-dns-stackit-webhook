package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/metrics"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/provider"
)

type Api interface {
	Listen(port string) error
	Test(req *http.Request, msTimeout ...int) (resp *http.Response, err error)
}

type api struct {
	logger *zap.Logger
	app    *fiber.App
}

func (a api) Test(req *http.Request, msTimeout ...int) (resp *http.Response, err error) {
	return a.app.Test(req, msTimeout...)
}

func (a api) Listen(port string) error {
	go func() {
		err := a.app.Listen(fmt.Sprintf(":%s", port))
		if err != nil {
			a.logger.Fatal("Error starting the server", zap.String(logFieldError, err.Error()))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh

	a.logger.Info(
		"shutting down server due to received signal",
		zap.String("signal", sig.String()),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := a.app.ShutdownWithContext(ctx)
	if err != nil {
		a.logger.Error("error shutting down server", zap.String("err", err.Error()))
	}

	cancel()

	return err
}

//go:generate mockgen -destination=./mock/provider.go github.com/stackitcloud/external-dns-stackit-webhook Provider
type Provider interface {
	provider.Provider
}

func New(logger *zap.Logger, middlewareCollector metrics.HttpApiMetrics, provider provider.Provider) Api {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
	})

	registerAt(app, "/metrics")
	app.Get("/healthz", Health)

	app.Use(NewMetricsMiddleware(middlewareCollector))
	app.Use(fiberlogger.New())
	app.Use(pprof.New(pprof.Config{Prefix: "/pprof"}))
	app.Use(fiberrecover.New())
	app.Use(helmet.New())

	webhookRoutes := webhook{
		provider: provider,
		logger:   logger,
	}

	app.Get("/records", webhookRoutes.Records)
	app.Post("/records", webhookRoutes.ApplyChanges)
	app.Post("/propertyvaluesequals", webhookRoutes.PropertyValuesEquals)
	app.Post("/adjustendpoints", webhookRoutes.AdjustEndpoints)

	return &api{
		logger: logger,
		app:    app,
	}
}
