package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/babylonchain/staking-api-service/internal/types"
)

const (
	verifyUTXOsPath = "/v1/ordinals/verify-utxos"
)

func TestVerifyUTXOs(t *testing.T) {
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	t.Run("Happy Path", func(t *testing.T) {
		// Define the test UTXOs
		utxos := []types.UTXORequest{
			{
				Txid: "6358dbafc9cfaa15a12f9624b1ad2c928c090fa05bff6219572361050bab4055",
				Vout: 0,
			},
		}

		// Marshal the request body
		requestBodyBytes, err := json.Marshal(utxos)
		require.NoError(t, err, "marshalling request body should not fail")

		// Make a POST request to the verify UTXOs endpoint
		verifyUTXOsUrl := testServer.Server.URL + verifyUTXOsPath
		resp, err := http.Post(verifyUTXOsUrl, "application/json", bytes.NewReader(requestBodyBytes))
		require.NoError(t, err, "making POST request to verify UTXOs endpoint should not fail")
		defer resp.Body.Close()

		// Check that the status code is HTTP 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

		// Read the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "reading response body should not fail")

		// Unmarshal the response body
		var response types.SafeUTXOResponse
		err = json.Unmarshal(bodyBytes, &response)
		require.NoError(t, err, "unmarshalling response body should not fail")

		// Check that the response data is as expected
		assert.NotEmpty(t, response.Data, "expected response data to be non-empty")
		assert.Equal(t, utxos[0].Txid, response.Data[0].TxId, "expected TxId to match")
		assert.Equal(t, false, response.Data[0].Inscription, "expected Inscription to be false")
		assert.Empty(t, response.Error, "expected no errors in the response")
	})

	t.Run("Invalid Input Format", func(t *testing.T) {
		// Define an invalid request body (not an array)
		invalidRequestBody := map[string]interface{}{
			"txid": "invalid_txid",
			"vout": 0,
		}

		// Marshal the invalid request body
		requestBodyBytes, err := json.Marshal(invalidRequestBody)
		require.NoError(t, err, "marshalling request body should not fail")

		// Make a POST request to the verify UTXOs endpoint
		verifyUTXOsUrl := testServer.Server.URL + verifyUTXOsPath
		resp, err := http.Post(verifyUTXOsUrl, "application/json", bytes.NewReader(requestBodyBytes))
		require.NoError(t, err, "making POST request to verify UTXOs endpoint should not fail")
		defer resp.Body.Close()

		// Check that the status code is HTTP 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected HTTP 400 Bad Request status")

		// Read the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "reading response body should not fail")

		// Unmarshal the response body
		var response types.SafeUTXOResponse
		err = json.Unmarshal(bodyBytes, &response)
		require.NoError(t, err, "unmarshalling response body should not fail")

		// Check that the response contains an error
		assert.NotEmpty(t, response.Error, "expected response error to be non-empty")
		require.Greater(t, len(response.Error), 0, "expected at least one error in the response")
		assert.Equal(t, "invalid input format", response.Error[0].Message, "expected error message to be 'invalid input format'")
		assert.Equal(t, http.StatusBadRequest, response.Error[0].Status, "expected error status to be 400")
		assert.Equal(t, "BAD_REQUEST", response.Error[0].ErrorCode, "expected error code to be 'BAD_REQUEST'")
	})

	t.Run("UTXO Not Found", func(t *testing.T) {
		// Define a UTXO that does not exist
		utxos := []types.UTXORequest{
			{
				Txid: "nonexistent_txid",
				Vout: 0,
			},
		}

		// Marshal the request body
		requestBodyBytes, err := json.Marshal(utxos)
		require.NoError(t, err, "marshalling request body should not fail")

		// Make a POST request to the verify UTXOs endpoint
		verifyUTXOsUrl := testServer.Server.URL + verifyUTXOsPath
		resp, err := http.Post(verifyUTXOsUrl, "application/json", bytes.NewReader(requestBodyBytes))
		require.NoError(t, err, "making POST request to verify UTXOs endpoint should not fail")
		defer resp.Body.Close()

		// Check that the status code is HTTP 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

		// Read the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "reading response body should not fail")

		// Unmarshal the response body
		var response types.SafeUTXOResponse
		err = json.Unmarshal(bodyBytes, &response)
		require.NoError(t, err, "unmarshalling response body should not fail")

		// Check that the response contains an error
		assert.NotEmpty(t, response.Error, "expected response error to be non-empty")
		require.Greater(t, len(response.Error), 0, "expected at least one error in the response")
		assert.Equal(t, "nonexistent_txid", response.Error[0].TxId, "expected TxId to match")
		assert.Equal(t, "UTXO not found.", response.Error[0].Message, "expected error message to be 'UTXO not found.'")
		assert.Equal(t, http.StatusNotFound, response.Error[0].Status, "expected error status to be 404")
		assert.Equal(t, "UTXO_NOT_FOUND", response.Error[0].ErrorCode, "expected error code to be 'UTXO_NOT_FOUND'")
	})

	t.Run("Multiple UTXOs", func(t *testing.T) {
		// Define multiple test UTXOs, including both valid and invalid ones
		utxos := []types.UTXORequest{
			{
				Txid: "6358dbafc9cfaa15a12f9624b1ad2c928c090fa05bff6219572361050bab4055",
				Vout: 0,
			},
			{
				Txid: "nonexistent_txid_1",
				Vout: 0,
			},
			{
				Txid: "nonexistent_txid_2",
				Vout: 1,
			},
		}

		// Marshal the request body
		requestBodyBytes, err := json.Marshal(utxos)
		require.NoError(t, err, "marshalling request body should not fail")

		// Make a POST request to the verify UTXOs endpoint
		verifyUTXOsUrl := testServer.Server.URL + verifyUTXOsPath
		resp, err := http.Post(verifyUTXOsUrl, "application/json", bytes.NewReader(requestBodyBytes))
		require.NoError(t, err, "making POST request to verify UTXOs endpoint should not fail")
		defer resp.Body.Close()

		// Check that the status code is HTTP 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

		// Read the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "reading response body should not fail")

		// Unmarshal the response body
		var response types.SafeUTXOResponse
		err = json.Unmarshal(bodyBytes, &response)
		require.NoError(t, err, "unmarshalling response body should not fail")

		// Check that the response data is as expected
		assert.Equal(t, 1, len(response.Data), "expected response data to contain 1 items")

		// Check that the error is as expected
		assert.Equal(t, 2, len(response.Error), "expected error to contain 2 items")

		// Validate the first UTXO (should be valid)
		assert.Equal(t, utxos[0].Txid, response.Data[0].TxId, "expected TxId to match")
		assert.Equal(t, false, response.Data[0].Inscription, "expected Inscription to be false")

		// Validate the second UTXO (should be invalid)
		assert.Equal(t, utxos[1].Txid, response.Error[0].TxId, "expected TxId to match")
		assert.Equal(t, "UTXO not found.", response.Error[0].Message, "expected error message to be 'UTXO not found.'")
		assert.Equal(t, http.StatusNotFound, response.Error[0].Status, "expected error status to be 404")
		assert.Equal(t, "UTXO_NOT_FOUND", response.Error[0].ErrorCode, "expected error code to be 'UTXO_NOT_FOUND'")

		// Validate the third UTXO (should be invalid)
		assert.Equal(t, utxos[2].Txid, response.Error[1].TxId, "expected TxId to match")
		assert.Equal(t, "UTXO not found.", response.Error[1].Message, "expected error message to be 'UTXO not found.'")
		assert.Equal(t, http.StatusNotFound, response.Error[1].Status, "expected error status to be 404")
		assert.Equal(t, "UTXO_NOT_FOUND", response.Error[1].ErrorCode, "expected error code to be 'UTXO_NOT_FOUND'")
	})
}
