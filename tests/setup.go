package tests

import (
	context "context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/babylonchain/staking-api-service/internal/api"
	"github.com/babylonchain/staking-api-service/internal/api/middlewares"
	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/db"
	"github.com/babylonchain/staking-api-service/internal/observability/metrics"
	"github.com/babylonchain/staking-api-service/internal/queue"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-queue-client/client"
	queueConfig "github.com/babylonchain/staking-queue-client/config"
	"github.com/go-chi/chi"
	"github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TestServerDependency struct {
	ConfigOverrides        *config.Config
	MockDbClient           db.DBClient
	PreInjectEventsHandler func(queueClient client.QueueClient) error
}

type TestServer struct {
	Server  *httptest.Server
	Queues  *queue.Queues
	Conn    *amqp091.Connection
	channel *amqp091.Channel
}

func (ts *TestServer) Close() {
	ts.Server.Close()
	ts.Queues.StopReceivingMessages()
	ts.Conn.Close()
	ts.channel.Close()
}

func setupTestServer(t *testing.T, dep *TestServerDependency) *TestServer {
	cfg, err := config.New("./config-test.yml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	metricsPort := cfg.Metrics.GetMetricsPort()
	metrics.Init(metricsPort)

	params, err := types.NewGlobalParams("./global-params-test.json")
	if err != nil {
		t.Fatalf("Failed to load global params: %v", err)
	}

	if dep != nil && dep.ConfigOverrides != nil {
		applyConfigOverrides(cfg, dep.ConfigOverrides)
	}

	services, err := services.New(context.Background(), cfg, params)
	if err != nil {
		t.Fatalf("Failed to initialize services: %v", err)
	}

	if dep != nil && dep.MockDbClient != nil {
		services.DbClient = dep.MockDbClient
	} else {
		// This means we are using real database, we not mocking anything
		setupTestDB(*cfg)
	}

	apiServer, err := api.New(context.Background(), cfg, services)
	if err != nil {
		t.Fatalf("Failed to initialize API server: %v", err)
	}

	// Setup routes
	r := chi.NewRouter()

	r.Use(middlewares.CorsMiddleware(cfg))
	apiServer.SetupRoutes(r)

	queues, conn, ch, err := setUpTestQueue(cfg.Queue, services)
	if err != nil {
		t.Fatalf("Failed to setup test queue: %v", err)
	}

	// Create an httptest server
	server := httptest.NewServer(r)

	return &TestServer{
		Server:  server,
		Queues:  queues,
		Conn:    conn,
		channel: ch,
	}
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

// PurgeAllCollections drops all collections in the specified database.
func PurgeAllCollections(ctx context.Context, client *mongo.Client, databaseName string) error {
	database := client.Database(databaseName)
	collections, err := database.ListCollectionNames(ctx, bson.D{{}})
	if err != nil {
		return err
	}

	for _, collection := range collections {
		if err := database.Collection(collection).Drop(ctx); err != nil {
			return err
		}
	}
	return nil
}

// setupTestDB connects to MongoDB and purges all collections.
func setupTestDB(cfg config.Config) *mongo.Client {
	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cfg.Db.Address))
	if err != nil {
		log.Fatal(err)
	}

	// Purge all collections in the test database
	if err := PurgeAllCollections(context.TODO(), client, cfg.Db.DbName); err != nil {
		log.Fatal("Failed to purge database:", err)
	}

	return client
}

func setUpTestQueue(cfg queueConfig.QueueConfig, service *services.Services) (*queue.Queues, *amqp091.Connection, *amqp091.Channel, error) {
	amqpURI := fmt.Sprintf("amqp://%s:%s@%s", cfg.QueueUser, cfg.QueuePassword, cfg.Url)
	conn, err := amqp091.Dial(amqpURI)
	if err != nil {
		log.Fatal("failed to connect to RabbitMQ in test: ", err)
		return nil, nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open a channel in test: %w", err)
	}
	purgeError := purgeQueues(ch, []string{
		client.ActiveStakingQueueName,
		client.UnbondingStakingQueueName,
		client.WithdrawStakingQueueName,
		client.ExpiredStakingQueueName,
	})
	if purgeError != nil {
		log.Fatal("failed to purge queues in test: ", purgeError)
		return nil, nil, nil, purgeError
	}

	// Start the actual queue processing in our codebase
	queues := queue.New(cfg, service)
	queues.StartReceivingMessages()

	return queues, conn, ch, nil
}

// inspectQueueMessageCount inspects the number of messages in the given queue.
func inspectQueueMessageCount(t *testing.T, conn *amqp091.Connection, queueName string) (int, error) {
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open a channel in test: %v", err)
	}
	q, err := ch.QueueInspect(queueName)
	if err != nil {
		if strings.Contains(err.Error(), "NOT_FOUND") || strings.Contains(err.Error(), "channel/connection is not open") {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to inspect queue in test %s: %w", queueName, err)
	}
	return q.Messages, nil
}

// purgeQueues purges all messages from the given list of queues.
func purgeQueues(ch *amqp091.Channel, queues []string) error {
	for _, queue := range queues {
		_, err := ch.QueuePurge(queue, false)
		if err != nil {
			if strings.Contains(err.Error(), "NOT_FOUND") || strings.Contains(err.Error(), "channel/connection is not open") {
				continue
			}
			return fmt.Errorf("failed to purge queue in test %s: %w", queue, err)
		}
	}

	return nil
}

func sendTestMessage[T any](client client.QueueClient, data []T) error {
	for _, d := range data {
		jsonBytes, err := json.Marshal(d)
		if err != nil {
			return err
		}
		messageBody := string(jsonBytes)
		err = client.SendMessage(context.TODO(), messageBody)
		if err != nil {
			return fmt.Errorf("failed to publish a message to queue %s: %w", client.GetQueueName(), err)
		}
	}
	return nil
}

// Inspect the items in the real database
func inspectDbDocuments[T any](t *testing.T, collectionName string) ([]T, error) {
	cfg, err := config.New("./config-test.yml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cfg.Db.Address))
	if err != nil {
		log.Fatal(err)
	}
	database := client.Database(cfg.Db.DbName)
	collection := database.Collection(collectionName)

	cursor, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var results []T
	for cursor.Next(context.Background()) {
		var result T
		err := cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func buildActiveStakingEvent(stakerHash string, numOfEvenet int) []client.ActiveStakingEvent {
	var activeStakingEvents []client.ActiveStakingEvent

	// To be replaced with https://github.com/babylonchain/staking-api-service/issues/21
	rand.New(rand.NewSource(time.Now().Unix()))

	for i := 0; i < numOfEvenet; i++ {
		activeStakingEvent := client.ActiveStakingEvent{
			EventType:             client.ActiveStakingEventType,
			StakingTxHashHex:      "0x1234567890abcdef" + fmt.Sprint(i),
			StakerPkHex:           stakerHash,
			FinalityProviderPkHex: "0xabcdef1234567890" + fmt.Sprint(i),
			StakingValue:          uint64(rand.Intn(1000)),
			StakingStartHeight:    uint64(rand.Intn(200)),
			StakingStartTimestamp: time.Now().String(),
			StakingTimeLock:       uint64(rand.Intn(100)),
			StakingOutputIndex:    uint64(rand.Intn(100)),
			StakingTxHex:          "0xabcdef1234567890" + fmt.Sprint(i),
		}
		activeStakingEvents = append(activeStakingEvents, activeStakingEvent)
	}
	return activeStakingEvents
}
