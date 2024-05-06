package tests

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/babylonchain/staking-api-service/internal/api/handlers"
	"github.com/babylonchain/staking-api-service/internal/services"
	"github.com/babylonchain/staking-api-service/internal/types"
	"github.com/babylonchain/staking-api-service/internal/utils"
)

const (
	globalParamsPath = "/v1/global-params"
)

func TestGlobalParams(t *testing.T) {
	testServer := setupTestServer(t, nil)
	defer testServer.Close()

	url := testServer.Server.URL + globalParamsPath

	// Make a GET request to the global params endpoint
	resp, err := http.Get(url)
	assert.NoError(t, err, "making GET request to global params endpoint should not fail")
	defer resp.Body.Close()

	// Check that the status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK status")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "reading response body should not fail")

	var responseBody handlers.PublicResponse[services.GlobalParamsPublic]
	err = json.Unmarshal(bodyBytes, &responseBody)
	assert.NoError(t, err, "unmarshalling response body should not fail")

	result := responseBody.Data.Versions
	// Check that the response body is as expected
	assert.NotEmpty(t, result, "expected response body to be non-empty")
	assert.Equal(t, 2, len(result))
	versionedGlobalParam := result[0]
	assert.Equal(t, uint64(0), versionedGlobalParam.Version)
	assert.Equal(t, uint64(100), versionedGlobalParam.ActivationHeight)
	assert.Equal(t, uint64(50), versionedGlobalParam.StakingCap)
	assert.Equal(t, "bbt4", versionedGlobalParam.Tag)
	assert.Equal(t, 5, len(versionedGlobalParam.CovenantPks))
	assert.Equal(t, uint64(3), versionedGlobalParam.CovenantQuorum)
	assert.Equal(t, uint64(1000), versionedGlobalParam.UnbondingTime)
	assert.Equal(t, uint64(10000), versionedGlobalParam.UnbondingFee)
	assert.Equal(t, uint64(300000), versionedGlobalParam.MaxStakingAmount)
	assert.Equal(t, uint64(3000), versionedGlobalParam.MinStakingAmount)
	assert.Equal(t, uint64(10000), versionedGlobalParam.MaxStakingTime)
	assert.Equal(t, uint64(100), versionedGlobalParam.MinStakingTime)

	versionedGlobalParam2 := result[1]
	assert.Equal(t, uint64(1), versionedGlobalParam2.Version)
	assert.Equal(t, uint64(200), versionedGlobalParam2.ActivationHeight)
	assert.Equal(t, uint64(500), versionedGlobalParam2.StakingCap)
	assert.Equal(t, "bbt4", versionedGlobalParam2.Tag)
	assert.Equal(t, 4, len(versionedGlobalParam2.CovenantPks))
	assert.Equal(t, uint64(2), versionedGlobalParam2.CovenantQuorum)
	assert.Equal(t, uint64(2000), versionedGlobalParam2.UnbondingTime)
	assert.Equal(t, uint64(20000), versionedGlobalParam2.UnbondingFee)
	assert.Equal(t, uint64(200000), versionedGlobalParam2.MaxStakingAmount)
	assert.Equal(t, uint64(2000), versionedGlobalParam2.MinStakingAmount)
	assert.Equal(t, uint64(20000), versionedGlobalParam2.MaxStakingTime)
	assert.Equal(t, uint64(200), versionedGlobalParam2.MinStakingTime)
}

var defaultParam = types.VersionedGlobalParams{
	Version:          0,
	ActivationHeight: 100,
	StakingCap:       50,
	Tag:              "bbt4",
	CovenantPks: []string{
		"03ffeaec52a9b407b355ef6967a7ffc15fd6c3fe07de2844d61550475e7a5233e5",
		"03a5c60c2188e833d39d0fa798ab3f69aa12ed3dd2f3bad659effa252782de3c31",
		"0359d3532148a597a2d05c0395bf5f7176044b1cd312f37701a9b4d0aad70bc5a4",
		"0357349e985e742d5131e1e2b227b5170f6350ac2e2feb72254fcc25b3cee21a18",
		"03c8ccb03c379e452f10c81232b41a1ca8b63d0baf8387e57d302c987e5abb8527",
	},
	CovenantQuorum:   3,
	UnbondingTime:    1000,
	UnbondingFee:     10000,
	MaxStakingAmount: 300000,
	MinStakingAmount: 3000,
	MaxStakingTime:   10000,
	MinStakingTime:   100,
}

func TestFailGlobalParamsValidation(t *testing.T) {
	var clonedParams types.GlobalParams
	defaultGlobalParams := types.GlobalParams{
		Versions: []*types.VersionedGlobalParams{&defaultParam},
	}
	// Empty versions
	jsonData := []byte(`{
		"versions": [
		]
	}`)
	fileName := createJsonFile(t, jsonData)

	// Call NewGlobalParams with the path to the temporary file
	_, err := types.NewGlobalParams(fileName)
	os.Remove(fileName)
	assert.Equal(t, "global params must have at least one version", err.Error())

	// invalid tag length
	utils.DeepCopy(&defaultGlobalParams, &clonedParams)
	clonedParams.Versions[0].Tag = "bbt"

	invalidTagJsonData, err := json.Marshal(clonedParams)
	assert.NoError(t, err, "marshalling invalid tag data should not fail")

	fileName = createJsonFile(t, invalidTagJsonData)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)
	assert.Equal(t, "invalid tag length, expected 4, got 3", err.Error())

	// test covenant pks sizes
	var invalidCovenantPksParam types.GlobalParams
	utils.DeepCopy(&defaultGlobalParams, &invalidCovenantPksParam)
	invalidCovenantPksParam.Versions[0].CovenantPks = []string{}

	invalidJson, err := json.Marshal(invalidCovenantPksParam)
	assert.NoError(t, err, "marshalling invalid covenant pks data should not fail")

	fileName = createJsonFile(t, invalidJson)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)
	assert.Equal(t, "empty covenant public keys", err.Error())

	// test covenant quorum
	utils.DeepCopy(&defaultGlobalParams, &clonedParams)
	clonedParams.Versions[0].CovenantQuorum = 6

	invalidJson, err = json.Marshal(clonedParams)
	assert.NoError(t, err, "marshalling invalid covenant pks data should not fail")

	fileName = createJsonFile(t, invalidJson)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)
	assert.Equal(t, "covenant quorum cannot be more than the amount of covenants", err.Error())

	// test invalid convenant pks
	utils.DeepCopy(&defaultGlobalParams, &clonedParams)
	clonedParams.Versions[0].CovenantPks = []string{
		"04ffeaec52a9b407b355ef6967a7ffc15fd6c3fe07de2844d61550475e7a5233e5",
		"03a5c60c2188e833d39d0fa798ab3f69aa12ed3dd2f3bad659effa252782de3c31",
		"0359d3532148a597a2d05c0395bf5f7176044b1cd312f37701a9b4d0aad70bc5a4",
		"0357349e985e742d5131e1e2b227b5170f6350ac2e2feb72254fcc25b3cee21a18",
		"03c8ccb03c379e452f10c81232b41a1ca8b63d0baf8387e57d302c987e5abb8527",
		"03c8ccb03c379e452f10c81232b41a1ca8b63d0baf8387e57d302c987e5abb8527",
	}

	jsonData, err = json.Marshal(clonedParams)
	assert.NoError(t, err, "marshalling invalid covenant pks data should not fail")

	fileName = createJsonFile(t, jsonData)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)
	assert.Contains(t, err.Error(), "invalid covenant public key")

	// test invalid min and max staking amount
	utils.DeepCopy(&defaultGlobalParams, &clonedParams)
	clonedParams.Versions[0].MaxStakingAmount = 300
	clonedParams.Versions[0].MinStakingAmount = 400

	jsonData, err = json.Marshal(clonedParams)
	assert.NoError(t, err, "marshalling invalid staking amount data should not fail")

	fileName = createJsonFile(t, jsonData)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)
	assert.Equal(t, "max-staking-amount must be larger than min-staking-amount", err.Error())

	// test activation height
	utils.DeepCopy(&defaultGlobalParams, &clonedParams)
	clonedParams.Versions[0].ActivationHeight = 0

	jsonData, err = json.Marshal(clonedParams)
	assert.NoError(t, err)

	fileName = createJsonFile(t, jsonData)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)
	assert.Equal(t, "activation height should be positive", err.Error())

	// test staking cap
	utils.DeepCopy(&defaultGlobalParams, &clonedParams)
	clonedParams.Versions[0].StakingCap = 0

	jsonData, err = json.Marshal(clonedParams)
	assert.NoError(t, err)

	fileName = createJsonFile(t, jsonData)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)
	assert.Equal(t, "staking cap should be positive", err.Error())
}

func TestGlobalParamsSortedByActivationHeight(t *testing.T) {
	params := generateGlobalParams(10)

	// We pick a random one and set its activation height to be less than its previous one
	params[5].ActivationHeight = params[4].ActivationHeight - 1

	globalParams := types.GlobalParams{
		Versions: params,
	}

	jsonData, err := json.Marshal(globalParams)
	assert.NoError(t, err)

	fileName := createJsonFile(t, jsonData)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)

	assert.Equal(t, "activation height cannot be overlapping between earlier and later versions", err.Error())
}

func TestGlobalParamsWithIncrementalVersions(t *testing.T) {
	params := generateGlobalParams(10)
	// We pick a random one and set its activation height to be less than its previous one
	params[5].Version = params[4].Version - 1

	globalParams := types.GlobalParams{
		Versions: params,
	}

	jsonData, err := json.Marshal(globalParams)
	assert.NoError(t, err)

	fileName := createJsonFile(t, jsonData)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)

	assert.Equal(t, "versions should be monotonically increasing by 1", err.Error())
}

func TestGlobalParamsWithIncrementalStakingCap(t *testing.T) {
	params := generateGlobalParams(10)
	// We pick a random one and set its activation height to be less than its previous one
	params[5].StakingCap = params[4].StakingCap - 1

	globalParams := types.GlobalParams{
		Versions: params,
	}

	jsonData, err := json.Marshal(globalParams)
	assert.NoError(t, err)

	fileName := createJsonFile(t, jsonData)
	_, err = types.NewGlobalParams(fileName)
	os.Remove(fileName)

	assert.Equal(t, "staking cap cannot be decreased in later versions", err.Error())
}

func generateGlobalParams(numOfParams int) []*types.VersionedGlobalParams {
	var params []*types.VersionedGlobalParams
	// generate a random number
	rand.Seed(time.Now().UnixNano())

	lastParam := defaultParam
	for i := 0; i < numOfParams; i++ {
		var param types.VersionedGlobalParams
		utils.DeepCopy(&defaultParam, &param)
		param.ActivationHeight = lastParam.ActivationHeight + uint64(rand.Intn(100))
		param.Version = uint64(i)
		param.StakingCap = lastParam.StakingCap + uint64(rand.Intn(100))
		params = append(params, &param)
		lastParam = param
	}

	return params
}
