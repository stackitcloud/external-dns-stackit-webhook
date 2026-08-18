package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/external-dns/endpoint"

	"github.com/stackitcloud/external-dns-stackit-webhook/internal/stackitprovider"
	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/metrics"
	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/stackit"
)

var (
	apiPort          string
	authBearerToken  string
	authKeyPath      string
	authWif          bool
	authWifTokenPath string
	tokenUrl         string
	baseUrl          string
	projectID        string
	worker           int
	domainFilter     []string
	dryRun           bool
	logLevel         string
)

var rootCmd = &cobra.Command{
	Use:   "external-dns-stackit-webhook",
	Short: "provider webhook for the STACKIT DNS service",
	Long:  "provider webhook for the STACKIT DNS service",
	Run: func(cmd *cobra.Command, args []string) {
		logger := getLogger()
		defer func(logger *zap.Logger) {
			err := logger.Sync()
			if err != nil {
				log.Println(err)
			}
		}(logger)

		endpointDomainFilter := endpoint.DomainFilter{Filters: domainFilter}

		authConfig := &stackit.WebhookAuthConfig{
			BaseURL:      baseUrl,
			TokenURL:     tokenUrl,
			Token:        authBearerToken,
			KeyPath:      authKeyPath,
			WIFEnabled:   authWif,
			WIFTokenPath: authWifTokenPath,
		}

		stackitConfigOptions, err := stackit.SetConfigOptions(authConfig)
		if err != nil {
			logger.Fatal("failed to set STACKIT config options", zap.Error(err))
		}

		stackitProvider, err := stackitprovider.NewStackitDNSProvider(
			logger.With(zap.String("component", "stackitprovider")),
			&stackitprovider.Config{
				ProjectId:    projectID,
				DomainFilter: endpointDomainFilter,
				DryRun:       dryRun,
				Workers:      worker,
			},
			stackitConfigOptions...,
		)
		if err != nil {
			logger.Fatal("failed to initialize STACKIT DNS provider", zap.Error(err))
		}

		app := api.New(logger.With(zap.String("component", "api")), metrics.NewHttpApiMetrics(), stackitProvider)
		err = app.Listen(apiPort)
		if err != nil {
			logger.Fatal("server error", zap.Error(err))
		}
	},
}

func getLogger() *zap.Logger {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(getZapLogLevel()),
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, errLogger := cfg.Build()
	if errLogger != nil {
		panic(errLogger)
	}

	return logger
}

func getZapLogLevel() zapcore.Level {
	switch logLevel {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&apiPort, "api-port", "8888", "Specifies the port to listen on.")
	rootCmd.PersistentFlags().StringVar(&authBearerToken, "auth-token", "", "Defines the authentication token for the STACKIT API. Mutually exclusive with 'auth-key-path' and 'auth-wif'.")
	rootCmd.PersistentFlags().StringVar(&authKeyPath, "auth-key-path", "", "Defines the file path of the service account key for the STACKIT API. Mutually exclusive with 'auth-token' and 'auth-wif'.")
	rootCmd.PersistentFlags().BoolVar(&authWif, "auth-wif", false, "Enables Workload Identity Federation (WIF) authentication explicitly.")
	rootCmd.PersistentFlags().StringVar(&authWifTokenPath, "auth-wif-token-path", "", "Defines a custom file path for the federated JWT token for WIF authentication.")
	rootCmd.PersistentFlags().StringVar(&tokenUrl, "token-url", "", "Defines the authentication token endpoint for the STACKIT API.")
	rootCmd.PersistentFlags().StringVar(&baseUrl, "base-url", "https://dns.api.stackit.cloud", "Identifies the Base URL for utilizing the API.")
	rootCmd.PersistentFlags().StringVar(&projectID, "project-id", "", "Specifies the project ID of the STACKIT project.")
	rootCmd.PersistentFlags().IntVar(&worker, "worker", 10, "Specifies the number of workers to employ for querying the API.")
	rootCmd.PersistentFlags().StringArrayVar(&domainFilter, "domain-filter", []string{}, "Establishes a filter for DNS zone names.")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Specifies whether to perform a dry run.")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Specifies the log level. Possible values are: debug, info, warn, error")

	err := rootCmd.MarkPersistentFlagRequired("project-id")
	if err != nil {
		panic(err)
	}
}

func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && viper.IsSet(f.Name) {
			if err := rootCmd.PersistentFlags().Set(f.Name, fmt.Sprint(viper.Get(f.Name))); err != nil {
				log.Fatalf("unable to set value for command line parameter: %v", err)
			}
		}
	})
}
