package runtimesecurity

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

const (
	EnabledEnv      = platform.SecurityEnabledEnv
	SecretNameEnv   = "SECURITY_SECRET_NAME"
	ProductTypeEnv  = "SECURITY_PRODUCT_TYPE"
	PollIntervalEnv = "SECURITY_POLL_INTERVAL"
	ConcurrencyEnv  = "SECURITY_CONCURRENCY"

	ServiceAccountEnv    = "SECURITY_SERVICE_ACCOUNT"
	ReaderClusterRoleEnv = "SECURITY_READER_CLUSTER_ROLE"

	defaultSecretName   = "ci-defectdojo"
	defaultProductType  = "KubeRocketCI"
	defaultPollInterval = 5 * time.Minute
	defaultConcurrency  = 2
)

// Config configures the runtime security controller.
type Config struct {
	// Enabled registers the controller; the feature is off by default.
	Enabled bool
	// SecretName is the Secret in the operator namespace with the DefectDojo url and token.
	SecretName string
	// ProductType is the DefectDojo product type the products are created in.
	ProductType string
	// PollInterval is the period between two passes over a Stage namespace.
	PollInterval time.Duration
	// Concurrency is the number of Stages processed at the same time.
	Concurrency int
	// ServiceAccount is the operator ServiceAccount, in the operator namespace, that reads Stage namespaces.
	ServiceAccount string
	// ReaderClusterRole is the ClusterRole with the reads of the controller, bound in each Stage namespace.
	ReaderClusterRole string
}

// ConfigFromEnv reads the configuration from the environment.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		SecretName:   stringEnv(SecretNameEnv, defaultSecretName),
		ProductType:  stringEnv(ProductTypeEnv, defaultProductType),
		PollInterval: defaultPollInterval,
		Concurrency:  defaultConcurrency,

		ServiceAccount:    os.Getenv(ServiceAccountEnv),
		ReaderClusterRole: os.Getenv(ReaderClusterRoleEnv),
	}

	if v := os.Getenv(EnabledEnv); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid %s %q: %w", EnabledEnv, v, err)
		}

		cfg.Enabled = enabled
	}

	if v := os.Getenv(PollIntervalEnv); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("invalid %s %q: a positive duration such as 5m is expected", PollIntervalEnv, v)
		}

		cfg.PollInterval = d
	}

	if v := os.Getenv(ConcurrencyEnv); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("invalid %s %q: a positive integer is expected", ConcurrencyEnv, v)
		}

		cfg.Concurrency = n
	}

	return cfg, nil
}

func stringEnv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}

	return fallback
}
