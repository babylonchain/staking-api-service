package tests

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"testing"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/clients"
	"github.com/babylonchain/staking-api-service/internal/clients/ordinals"
	"github.com/babylonchain/staking-api-service/internal/config"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
	"github.com/babylonchain/staking-api-service/tests/mocks"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const verifyUTXOsPath = "/v1/ordinals/verify-utxos"

func createPayload(t *testing.T, r *rand.Rand, netParam *chaincfg.Params, size int) handlers.VerifyUTXOsRequestPayload {
	var utxos []types.UTXOIdentifier
	for i := 0; i < size; i++ {
		tx, _, err := generateRandomTx(r)
		if err != nil {
			t.Fatalf("Failed to generate random tx: %v", err)
		}
		utxos = append(utxos, types.UTXOIdentifier{
			Txid: tx.TxHash().String(),
			Vout: uint32(r.Intn(10)),
		})
	}
	pk, err := randomPk()
	if err != nil {
		t.Fatalf("Failed to generate random pk: %v", err)
	}
	address, err := utils.GetTaprootAddressFromPk(pk, netParam)
	if err != nil {
		t.Fatalf("Failed to generate taproot address from pk: %v", err)
	}
	return handlers.VerifyUTXOsRequestPayload{
		UTXOs:   utxos,
		Address: address,
	}
}

func TestVerifyUtxosEndpointNotAvailableIfAssetsConfigNotSet(t *testing.T) {
	cfg, err := config.New("./config/config-test.yml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	cfg.Assets = nil

	testServer := setupTestServer(t, &TestServerDependency{ConfigOverrides: cfg})
	defer testServer.Close()

	url := testServer.Server.URL + verifyUTXOsPath
	resp, err := http.Post(url, "application/json", bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("Failed to make POST request to %s: %v", url, err)
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func FuzzSuccessfullyVerifyUTXOsAssetsViaOrdinalService(f *testing.F) {
	attachRandomSeedsToFuzzer(f, 10)
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))
		numOfUTXOs := randomPositiveInt(r, 100)
		payload := createPayload(t, r, &chaincfg.MainNetParams, numOfUTXOs)
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		// create some ordinal responses that contains inscriptions
		numOfUTXOsWithAsset := r.Intn(numOfUTXOs)

		var txidsWithAsset []string
		for i := 0; i < numOfUTXOsWithAsset; i++ {
			txidsWithAsset = append(txidsWithAsset, payload.UTXOs[i].Txid)
		}

		mockedOrdinalResponse := createOrdinalServiceResponse(t, r, payload.UTXOs, txidsWithAsset)

		mockOrdinal := new(mocks.OrdinalsClientInterface)
		mockOrdinal.On("FetchUTXOInfos", mock.Anything, mock.Anything).Return(mockedOrdinalResponse, nil)
		mockedClients := &clients.Clients{
			Ordinals: mockOrdinal,
		}
		testServer := setupTestServer(t, &TestServerDependency{MockedClients: mockedClients})
		defer testServer.Close()

		url := testServer.Server.URL + verifyUTXOsPath
		resp, err := http.Post(url, "application/json", bytes.NewReader(jsonPayload))
		if err != nil {
			t.Fatalf("Failed to make POST request to %s: %v", url, err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// decode the response body
		var response handlers.PublicResponse[[]services.SafeUTXOPublic]
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response body: %v", err)
		}

		// check the response
		assert.Equal(t, len(payload.UTXOs), len(response.Data))
		// check if the inscriptions are correctly returned and order is preserved
		for i, u := range response.Data {
			// Make sure the UTXO identifiers are correct
			assert.Equal(t, payload.UTXOs[i].Txid, u.TxId)
			assert.Equal(t, payload.UTXOs[i].Vout, u.Vout)
			var isWithAsset bool
			for _, txid := range txidsWithAsset {
				if txid == u.TxId {
					assert.True(t, u.Inscription)
					isWithAsset = true
					break
				}
			}
			if !isWithAsset {
				assert.False(t, u.Inscription)
			}
		}
	})
}

// TODO: Test case 2: Fetching more than 100 UTXOs should return an error
// TODO: Test case 3: Invalid UTXO txid should return an error
// TODO: Test case 4: Ordinal service return error, fallback to unisat service and return the result
// TODO: Test case 5: Unisat service return error, return error
// TODO: Test case 6: Ordinal service took too long to respond, fallback to unisat service and return the result
// TODO: Test case 7: Unisat service took too long to respond, return error within the timeout window
// TODO: Test case 8: Send 100 UTXOs, ordinal service fail, unisat service return 100(limit), then fetch again for the remaining 1 item (test the pagination)
// TODO: Test case 9: Fall back to unisat when ordinal service return data that is not in the right order of the request UTXOs

func createOrdinalServiceResponse(t *testing.T, r *rand.Rand, utxos []types.UTXOIdentifier, txidsWithAsset []string) []ordinals.OrdinalsOutputResponse {
	var responses []ordinals.OrdinalsOutputResponse

	for _, utxo := range utxos {
		withAsset := false
		for _, txid := range txidsWithAsset {
			if txid == utxo.Txid {
				withAsset = true
				break
			}
		}
		if withAsset {
			// randomly inject runes or inscriptions
			if r.Intn(2) == 0 {
				responses = append(responses, ordinals.OrdinalsOutputResponse{
					Transaction:  utxo.Txid,
					Inscriptions: []string{randomString(r, r.Intn(100))},
					Runes:        json.RawMessage(`{}`),
				})
			} else {
				responses = append(responses, ordinals.OrdinalsOutputResponse{
					Transaction:  utxo.Txid,
					Inscriptions: []string{},
					Runes:        json.RawMessage(`{"rune1": "rune1"}`),
				})
			}
		} else {
			responses = append(responses, ordinals.OrdinalsOutputResponse{
				Transaction:  utxo.Txid,
				Inscriptions: []string{},
				Runes:        json.RawMessage(`{}`),
			})
		}

	}
	return responses
}
