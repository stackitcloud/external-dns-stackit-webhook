package cmd

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
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
	apiPort              string
	authBearerToken      string
	baseUrl              string
	projectID            string
	worker               int
	domainFilter         []string
	domainExclude        []string
	domainRegex          string
	domainRegexExclusion string
)

var rootCmd = &cobra.Command{
	Use:   "external-dns-stackit-webhook",
	Short: "provider webhook for the STACKIT DNS service",
	Long:  "provider webhook for the STACKIT DNS service",
	Run: func(cmd *cobra.Command, args []string) {
		logger, errLogger := zap.NewProduction()
		if errLogger != nil {
			panic(errLogger)
		}
		defer func(logger *zap.Logger) {
			err := logger.Sync()
			if err != nil {
				log.Println(err)
			}
		}(logger)

		endpointDomainFilter := endpoint.DomainFilter{}
		if domainRegex != "" {
			endpointDomainFilter = endpoint.NewRegexDomainFilter(regexp.MustCompile(domainRegex), regexp.MustCompile(domainRegexExclusion))
		} else {
			endpointDomainFilter.Filters = domainFilter
			endpointDomainFilter = endpoint.NewDomainFilterWithExclusions(endpointDomainFilter.Filters, domainExclude)
		}

		stackitProvider, err := stackitprovider.NewStackitDNSProvider(stackitprovider.Config{
			BasePath:     baseUrl,
			Token:        authBearerToken,
			ProjectId:    projectID,
			DomainFilter: endpointDomainFilter,
			DryRun:       false,
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

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&apiPort, "api-port", "8888", "Port to listen on for the API")
	rootCmd.PersistentFlags().StringVar(&authBearerToken, "auth-bearer-token", "", "Bearer token to use for authentication")
	rootCmd.PersistentFlags().StringVar(&baseUrl, "base-url", "http://localhost:3000", "Base URL to use for the API")
	rootCmd.PersistentFlags().StringVar(&projectID, "project-id", "", "Project to use for the API")
	rootCmd.PersistentFlags().IntVar(&worker, "worker", 10, "Number of workers to use for querying the API. "+
		"Since we have to iterate over all zones and records we can parallelize it. But keep in mind to not set it too high "+
		"since you will receive 429 rate limiting from the API")
	rootCmd.PersistentFlags().StringArrayVar(&domainFilter, "domain-filter", []string{}, "Filter to apply to DNS records. Only this filter or the domain regex filter can be used at the same time")
	rootCmd.PersistentFlags().StringArrayVar(&domainExclude, "exclude-domain-filter", []string{}, "Filter to exclude DNS records")
	rootCmd.PersistentFlags().StringVar(&domainRegex, "domain-regex-filter", "", "Regex to apply to DNS records. Only this filter or the domain filter can be used at the same time")
	rootCmd.PersistentFlags().StringVar(&domainRegexExclusion, "exclude-domain-regex-filter", "", "Regex to exclude DNS records")
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
