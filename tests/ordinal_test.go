package tests

import (
	"testing"
)

const (
	verifyUTXOsPath = "/v1/ordinals/verify-utxos"
)

func TestVerifyUTXOs(t *testing.T) {
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	// happy paths
	t.Run("Single UTXO", func(t *testing.T) {
	})

	t.Run("Multiple UTXOs", func(t *testing.T) {
	})

	// error paths

	t.Run("Invalid Input Format", func(t *testing.T) {
	})

	t.Run("UTXO Not Found", func(t *testing.T) {
	})
}