package tests

import (
	context "context"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/babylonchain/staking-api-service/internal/api"
	"github.com/babylonchain/staking-api-service/internal/api/middlewares"
	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/observability/metrics"
	"github.com/babylonchain/staking-api-service/internal/services"
	testmock "github.com/babylonchain/staking-api-service/tests/mocks"
	"github.com/go-chi/chi"
)

type TestServerDependency struct {
	ConfigOverrides *config.Config
	DBClient        db.DBClient
}

func setupTestServer(t *testing.T, dep *TestServerDependency) *httptest.Server {
	cfg, err := config.New("./config-test.yml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	metricsPort := cfg.Metrics.GetMetricsPort()
	metrics.Init(metricsPort)

	if dep == nil {
		dep = &TestServerDependency{
			DBClient: new(testmock.DBClient),
		}
	}
	if dep.DBClient == nil {
		dep.DBClient = new(testmock.DBClient)
	}

	if dep.ConfigOverrides != nil {
		applyConfigOverrides(cfg, dep.ConfigOverrides)
	}

	services, err := services.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to initialize services: %v", err)
	}
	services.DbClient = dep.DBClient

	apiServer, err := api.New(context.Background(), cfg, services)
	if err != nil {
		t.Fatalf("Failed to initialize API server: %v", err)
	}

	// Setup routes
	r := chi.NewRouter()

	r.Use(middlewares.CorsMiddleware(cfg))
	apiServer.SetupRoutes(r)

	// Create an httptest server
	server := httptest.NewServer(r)

	return server
}

// Generic function to apply configuration overrides
func applyConfigOverrides(defaultCfg *config.Config, overrides *config.Config) {
	defaultVal := reflect.ValueOf(defaultCfg).Elem()
	overrideVal := reflect.ValueOf(overrides).Elem()

	for i := 0; i < defaultVal.NumField(); i++ {
		defaultField := defaultVal.Field(i)
		overrideField := overrideVal.Field(i)

		if overrideField.IsZero() {
			continue // Skip fields that are not set
		}

		if defaultField.CanSet() {
			defaultField.Set(overrideField)
		}
	}
}
