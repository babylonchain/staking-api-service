package tests

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/babylonchain/staking-api-service/internal/clients"
	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/services"
)

const (
	verifyUTXOsPath = "/v1/ordinals/verify-utxos"
)

func TestVerifyUTXOs(t *testing.T) {
	cfg, err := config.New("./config/config-test.yml")
	require.NoError(t, err)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mockPayload, err := generateRandomVerifyUTXOsRequestPayload(cfg, rng, 1, 10)
	require.NoError(t, err)

	mockOrdinalService := mockOrdinalService(verifyUTXOsPath, mockPayload, rng, 0, 0)
	defer mockOrdinalService.Close()

	testServer := setupTestServer(t, &TestServerDependency{
		MockClients: clients.New(cfg),
	})
	defer testServer.Close()

	url := mockOrdinalService.URL + verifyUTXOsPath
	requestBody, err := json.Marshal(mockPayload)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result []services.SafeUTXOPublic
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result, len(mockPayload.Utxos))
	for i, utxo := range mockPayload.Utxos {
		assert.Equal(t, utxo.Txid, result[i].TxId)
	}
}

func TestVerifyUTXOs_OrdinalFailedUnisatSuccess(t *testing.T) {
	cfg, err := config.New("./config/config-test.yml")
	require.NoError(t, err)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mockPayload, err := generateRandomVerifyUTXOsRequestPayload(cfg, rng, 1, 10)
	require.NoError(t, err)

	mockOrdinalService := mockOrdinalService(verifyUTXOsPath, mockPayload, rng, http.StatusInternalServerError, 0)
	defer mockOrdinalService.Close()

	url := mockOrdinalService.URL + verifyUTXOsPath
	requestBody, err := json.Marshal(mockPayload)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestVerifyUTXOs_OrdinalTimeoutUnisatSuccess(t *testing.T) {
	cfg, err := config.New("./config/config-test.yml")
	require.NoError(t, err)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mockPayload, err := generateRandomVerifyUTXOsRequestPayload(cfg, rng, 1, 10)
	require.NoError(t, err)

	mockOrdinalService := mockOrdinalService(verifyUTXOsPath, mockPayload, rng, http.StatusInternalServerError, 0)
	defer mockOrdinalService.Close()

	url := mockOrdinalService.URL + verifyUTXOsPath
	requestBody, err := json.Marshal(mockPayload)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestVerifyUTXOs_OrdinalAndUnisatTimeout(t *testing.T) {
	cfg, err := config.New("./config/config-test.yml")
	require.NoError(t, err)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mockPayload, err := generateRandomVerifyUTXOsRequestPayload(cfg, rng, 1, 10)
	require.NoError(t, err)

	mockOrdinalService := mockOrdinalService(verifyUTXOsPath, mockPayload, rng, http.StatusInternalServerError, http.StatusInternalServerError)
	defer mockOrdinalService.Close()

	url := mockOrdinalService.URL + verifyUTXOsPath
	requestBody, err := json.Marshal(mockPayload)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "expected HTTP 500 Internal Service Error status code")

}

func TestVerifyUTXOs_InputSyntaxError(t *testing.T) {
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	url := testServer.Server.URL + verifyUTXOsPath
	invalidPayload := `{"invalid": "input"}`

	resp, err := http.Post(url, "application/json", bytes.NewReader([]byte(invalidPayload)))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestVerifyUTXOs_NoConfig(t *testing.T) {
	cfg, err := config.New("./config/config-test.yml")
	require.NoError(t, err)

	testServer := setupTestServer(t, &TestServerDependency{
		ConfigOverrides: &config.Config{
			Assets: nil,
		},
	})
	defer testServer.Close()

	url := testServer.Server.URL + verifyUTXOsPath
	payload, err := generateRandomVerifyUTXOsRequestPayload(cfg, rand.New(rand.NewSource(time.Now().UnixNano())), 1, 10)
	require.NoError(t, err)
	requestBody, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestVerifyUTXOs_OrdinalConfigPresentUnisatNot(t *testing.T) {
	cfg, err := config.New("./config/config-test.yml")
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mockPayload, err := generateRandomVerifyUTXOsRequestPayload(cfg, rng, 1, 10)
	require.NoError(t, err)

	mockOrdinalService := mockOrdinalService(verifyUTXOsPath, mockPayload, rng, 0, 0)
	defer mockOrdinalService.Close()

	testServer := setupTestServer(t, &TestServerDependency{
		ConfigOverrides: &config.Config{
			Assets: &config.AssetsConfig{
				MaxUTXOs: cfg.Assets.MaxUTXOs,
				Ordinals: cfg.Assets.Ordinals,
				Unisat:   nil,
			},
		},
	})
	defer testServer.Close()

	url := mockOrdinalService.URL + verifyUTXOsPath
	requestBody, err := json.Marshal(mockPayload)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(requestBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
