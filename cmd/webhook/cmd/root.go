package cmd

import (
	"fmt"
	"go.uber.org/zap/zapcore"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stackitcloud/external-dns-stackit-webhook/internal/stackitprovider"
	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/metrics"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
)

var (
	apiPort         string
	authBearerToken string
	baseUrl         string
	projectID       string
	worker          int
	domainFilter    []string
	dryRun          bool
	logLevel        string
)

var rootCmd = &cobra.Command{
	Use:   "external-dns-stackit-webhook",
	Short: "provider webhook for the STACKIT DNS service",
	Long:  "provider webhook for the STACKIT DNS service",
	Run: func(cmd *cobra.Command, args []string) {
		if len(authBearerToken) == 0 {
			panic("auth-token is required")
		}

		logger := getLogger()
		defer func(logger *zap.Logger) {
			err := logger.Sync()
			if err != nil {
				log.Println(err)
			}
		}(logger)

		endpointDomainFilter := endpoint.DomainFilter{Filters: domainFilter}

		stackitProvider, err := stackitprovider.NewStackitDNSProvider(stackitprovider.Config{
			BasePath:     baseUrl,
			Token:        authBearerToken,
			ProjectId:    projectID,
			DomainFilter: endpointDomainFilter,
			DryRun:       dryRun,
			Workers:      worker,
		}, logger.With(zap.String("component", "stackitprovider")), &http.Client{
			Timeout: 10 * time.Second,
		})
		if err != nil {
			panic(err)
		}

		app := api.New(logger.With(zap.String("component", "api")), metrics.NewHttpApiMetrics(), stackitProvider)
		err = app.Listen(apiPort)
		if err != nil {
			panic(err)
		}
	},
}

func getLogger() *zap.Logger {
	cfg := zap.Config{
		Level:    zap.NewAtomicLevelAt(getZapLogLevel()),
		Encoding: "json", // or "console"
		// ... other zap configuration as needed
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
	case "DEBUG":
		return zapcore.DebugLevel
	case "INFO":
		return zapcore.InfoLevel
	case "WARN":
		return zapcore.WarnLevel
	case "ERROR":
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
	rootCmd.PersistentFlags().StringVar(&authBearerToken, "auth-token", "", "Defines the authentication token for the STACKIT API.")
	rootCmd.PersistentFlags().StringVar(&baseUrl, "base-url", "https://dns.api.stackit.cloud", " Identifies the Base URL for utilizing the API.")
	rootCmd.PersistentFlags().StringVar(&projectID, "project-id", "", "Specifies the project id of the STACKIT project.")
	rootCmd.PersistentFlags().IntVar(&worker, "worker", 10, "Specifies the number "+
		"of workers to employ for querying the API. Given that we need to iterate over all zones and "+
		"records, it can be parallelized. However, it is important to avoid setting this number "+
		"excessively high to prevent receiving 429 rate limiting from the API.")
	rootCmd.PersistentFlags().StringArrayVar(&domainFilter, "domain-filter", []string{}, "Establishes a filter for DNS zone names")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Specifies whether to perform a dry run.")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "INFO", "Specifies the log level. Possible values are: DEBUG, INFO, WARN, ERROR")
}

func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// There is some issue, where the integration of Cobra with Viper will result in wrong values, therefore we are
	// setting the values from viper manually. The issue is, that with the standard integration, viper will see, that
	// Cobra parameters are set - even if the command line parameter was not used and the default value was set. But
	// when Viper notices that the value is set, it will not overwrite the default value with the environment variable.
	// Another possibility would be to not have any default values set for cobra command line parameters, but this would
	// break the automatic help output from the cli. The manual way here seems the best solution for now.
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && viper.IsSet(f.Name) {
			if err := rootCmd.PersistentFlags().Set(f.Name, fmt.Sprint(viper.Get(f.Name))); err != nil {
				log.Fatalf("unable to set value for command line parameter: %v", err)
			}
		}
	})
}
