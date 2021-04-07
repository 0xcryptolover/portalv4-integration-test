package awesomeProject1

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

const BTC_DECIMAL = 1e8
const BURN_ADDRESS = "15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs"

type RPCErrorInc struct {
	Code    int    `json:"Code"`
	Message string `json:"Message"`
}

type RPCBaseRes struct {
	Id       int          `json:"Id"`
	RPCError *RPCErrorInc `json:"Error"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	Result interface{} `json:"result"`
	Id     string      `json:"id"`
	Error  *RPCError   `json:"error"`
}

type ResponseInc struct {
	Result interface{} `json:"Result"`
	RPCBaseRes
}

type MerkleProof struct {
	ProofHash *chainhash.Hash
	IsLeft    bool
}

type BTCProof struct {
	MerkleProofs []*MerkleProof
	BTCTx        *wire.MsgTx
	BlockHash    *chainhash.Hash
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including assertion methods.
type PortalV4BaseTestSuite struct {
	suite.Suite
	IncBurningAddrStr string
	IncPrivKeyStr     string
	IncPaymentAddrStr string
	IncBridgeHost     string
}

// Make sure that VariableThatShouldStartAtFive is set to five
// before each test
func (portalV4Suite *PortalV4BaseTestSuite) SetupSuite() {
	fmt.Println("Setting up the suite...")

	portalV4Suite.IncBurningAddrStr = "15pABFiJVeh9D5uiQEhQX4SVibGGbdAVipQxBdxkmDqAJaoG1EdFKHBrNfs"
	portalV4Suite.IncPrivKeyStr = "112t8roafGgHL1rhAP9632Yef3sx5k8xgp8cwK4MCJsCL1UWcxXvpzg97N4dwvcD735iKf31Q2ZgrAvKfVjeSUEvnzKJyyJD3GqqSZdxN4or"
	portalV4Suite.IncPaymentAddrStr = "12S5Lrs1XeQLbqN4ySyKtjAjd2d7sBP2tjFijzmp6avrrkQCNFMpkXm3FPzj2Wcu2ZNqJEmh9JriVuRErVwhuQnLmWSaggobEWsBEci"
	portalV4Suite.IncBridgeHost = "http://127.0.0.1:9334"
}

func (portalV4Suite *PortalV4BaseTestSuite) TearDownSuite() {
	fmt.Println("Tearing down the suite...")
}

func (portalV4Suite *PortalV4BaseTestSuite) SetupTest() {
	fmt.Println("Setting up the test...")
}

func (portalV4Suite *PortalV4BaseTestSuite) TearDownTest() {
	fmt.Println("Tearing down the test...")
}

func (portalV4Suite *PortalV4BaseTestSuite) TestPortalV4BaseTestSuite() {
	fmt.Println("This is generic test suite")
}

func wait(client *ethclient.Client, tx common.Hash) error {
	ctx := context.Background()
	for range time.Tick(10 * time.Second) {
		_, err := client.TransactionReceipt(ctx, tx)
		if err == nil {
			break
		} else if err == ethereum.NotFound {
			continue
		} else {
			return err
		}
	}
	return nil
}

//func getPortalCustodianWithdrawV3(url, txHash, rpcMethod string) (map[string]interface{}, error) {
//	rpcClient := rpccaller.NewRPCClient()
//	transactionId := map[string]interface{}{"TxId": txHash}
//	inputParams := []interface{}{transactionId}
//	var res CommonRes
//	err := rpcClient.RPCCall(
//		"",
//		url,
//		"",
//		rpcMethod,
//		inputParams,
//		&res,
//	)
//	if err != nil || res.Result == nil {
//		return nil, err
//	}
//	return res.Result.(map[string]interface{}), nil
//}

func createAndSendRawTx(client *rpcclient.Client, host, user string, from btcutil.Address, inputs []btcjson.TransactionInput, outputs []map[string]interface{}) (*chainhash.Hash, error) {
	outputsJson, err := json.Marshal(outputs)
	if err != nil {
		return nil, err
	}
	inputsJson, err := json.Marshal(inputs)
	if err != nil {
		return nil, err
	}
	createRawTx := fmt.Sprintf("curl -u %v --data-binary '{\"jsonrpc\": \"1.0\", \"id\":\"curltest\", \"method\": \"createrawtransaction\", \"params\": [%v, %v] }' -H 'content-type: text/plain;' %v", user, string(inputsJson), string(outputsJson), host)
	rep, err := exec.Command("/bin/sh", "-c", createRawTx).Output()
	if err != nil {
		return nil, err
	}
	var response Response
	err = json.Unmarshal(rep, &response)
	if err != nil {
		return nil, err
	}

	priv, err := client.DumpPrivKey(from)
	if err != nil {
		return nil, err
	}

	signedRawTxResult := fmt.Sprintf("curl -u %v --data-binary '{\"jsonrpc\": \"1.0\", \"id\":\"curltest\", \"method\": \"signrawtransactionwithkey\", \"params\": [\"%v\", [\"%v\"]] }' -H 'content-type: text/plain;' %v", user, response.Result.(string), priv.String(), host)
	repTxSigned, err := exec.Command("/bin/sh", "-c", signedRawTxResult).Output()
	err = json.Unmarshal(repTxSigned, &response)
	if err != nil || response.Result == nil {
		return nil, err
	}
	hexRawTx := response.Result.(map[string]interface{})["hex"].(string)
	hexRawTxBytes, err := hex.DecodeString(hexRawTx)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewReader(hexRawTxBytes)
	redeemTx := wire.NewMsgTx(wire.TxVersion)
	err = redeemTx.Deserialize(buffer)
	if err != nil {
		return nil, err
	}

	txid, err := client.SendRawTransaction(redeemTx, true)
	if err != nil {
		return nil, err
	}
	return txid, nil
}

func buildProof(client *rpcclient.Client, txHash *chainhash.Hash) string {
	msgTx := buildMsgTx(txHash, client)
	txInfo, err := client.GetTransaction(txHash)
	if err != nil {
		fmt.Printf("Could not get tx info with err: %v \n", err)
		return ""
	}

	blockHash, err := chainhash.NewHashFromStr(txInfo.BlockHash)
	if err != nil {
		fmt.Printf("Could decode blockhash with err: %v \n", err)
		return ""
	}
	blockInfo, err := client.GetBlock(blockHash)
	if err != nil {
		fmt.Printf("Could not get block by hash: %v \n", err)
		return ""
	}

	txs := blockInfo.Transactions
	txHashes := make([]*chainhash.Hash, len(txs))
	for i := range txs {
		temp := txs[i].TxHash()
		txHashes[i] = &temp
	}

	txHashNew := msgTx.TxHash()
	blkHash, _ := chainhash.NewHashFromStr(txInfo.BlockHash)
	merkleProofs := buildMerkleProof(txHashes, &txHashNew)
	btcProof := BTCProof{
		MerkleProofs: merkleProofs,
		BTCTx:        msgTx,
		BlockHash:    blkHash,
	}
	btcProofBytes, _ := json.Marshal(btcProof)
	return base64.StdEncoding.EncodeToString(btcProofBytes)
}

func buildMsgTx(txHash *chainhash.Hash, client *rpcclient.Client) *wire.MsgTx {
	txInfo, err := client.GetTransaction(txHash)
	if err != nil {
		fmt.Printf("Can not get transaction with id: %v", txHash.String())
		return nil
	}
	temp, _ := hex.DecodeString(txInfo.Hex)
	buffer := bytes.NewReader(temp)
	msgTx := wire.NewMsgTx(wire.TxVersion)
	err = msgTx.Deserialize(buffer)
	if err != nil {
		fmt.Printf("Can not deserialize transaction with raw tx: %v", txInfo.Hex)
		return nil
	}

	return msgTx
}

func buildMerkleProof(txHashes []*chainhash.Hash, targetedTxHash *chainhash.Hash) []*MerkleProof {
	merkleTree := buildMerkleTreeStoreFromTxHashes(txHashes)
	nextPoT := nextPowerOfTwo(len(txHashes))
	layers := [][]*chainhash.Hash{}
	left := 0
	right := nextPoT
	for left < right {
		layers = append(layers, merkleTree[left:right])
		curLen := len(merkleTree[left:right])
		left = right
		right = right + curLen/2
	}

	merkleProofs := []*MerkleProof{}
	curHash := targetedTxHash
	for _, layer := range layers {
		if len(layer) == 1 {
			break
		}

		for i := 0; i < len(layer); i++ {
			if layer[i] == nil || layer[i].String() != curHash.String() {
				continue
			}
			if i%2 == 0 {
				if layer[i+1] == nil {
					curHash = HashMerkleBranches(layer[i], layer[i])
					merkleProofs = append(
						merkleProofs,
						&MerkleProof{
							ProofHash: layer[i],
							IsLeft:    false,
						},
					)
				} else {
					curHash = HashMerkleBranches(layer[i], layer[i+1])
					merkleProofs = append(
						merkleProofs,
						&MerkleProof{
							ProofHash: layer[i+1],
							IsLeft:    false,
						},
					)
				}
			} else {
				if layer[i-1] == nil {
					curHash = HashMerkleBranches(layer[i], layer[i])
					merkleProofs = append(
						merkleProofs,
						&MerkleProof{
							ProofHash: layer[i],
							IsLeft:    true,
						},
					)
				} else {
					curHash = HashMerkleBranches(layer[i-1], layer[i])
					merkleProofs = append(
						merkleProofs,
						&MerkleProof{
							ProofHash: layer[i-1],
							IsLeft:    true,
						},
					)
				}
			}
			break // process next layer
		}
	}
	return merkleProofs
}

func buildMerkleTreeStoreFromTxHashes(txHashes []*chainhash.Hash) []*chainhash.Hash {
	nextPoT := nextPowerOfTwo(len(txHashes))
	arraySize := nextPoT*2 - 1
	merkles := make([]*chainhash.Hash, arraySize)

	for i, txHash := range txHashes {
		merkles[i] = txHash
	}

	offset := nextPoT
	for i := 0; i < arraySize-1; i += 2 {
		switch {
		case merkles[i] == nil:
			merkles[offset] = nil

		case merkles[i+1] == nil:
			newHash := HashMerkleBranches(merkles[i], merkles[i])
			merkles[offset] = newHash

		default:
			newHash := HashMerkleBranches(merkles[i], merkles[i+1])
			merkles[offset] = newHash
		}
		offset++
	}

	return merkles
}

func shieldToken(rpcMethod, url, proof, incPrivKeyStr, paymentAddress, tokenID string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"IncogAddressStr": paymentAddress,
		"TokenID":         tokenID,
		"ShieldingProof":  proof,
	}
	params := []interface{}{
		incPrivKeyStr,
		nil,
		5,
		-1,
		meta,
		"",
		0,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling shield ptokens err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

func getShieldStatus(txHash, url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"ReqTxID": txHash,
	}
	params := []interface{}{
		meta,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

func unshieldPToken(rpcMethod, url, incPrivKeyStr, paymentAddress, externalAddress, tokenID string, amount string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	tokenRecieve := map[string]interface{}{
		BURN_ADDRESS: amount,
	}

	meta := map[string]interface{}{
		"Privacy":        true,
		"TokenID":        tokenID,
		"TokenTxType":    1,
		"TokenName":      "",
		"TokenSymbol":    "",
		"TokenAmount":    amount,
		"TokenReceivers": tokenRecieve,
		"TokenFee":       "0",

		"PortalTokenID":  tokenID,
		"UnshieldAmount": amount,
		"IncAddressStr":  paymentAddress,
		"RemoteAddress":  externalAddress,
	}
	params := []interface{}{
		incPrivKeyStr,
		nil,
		-1,
		-1,
		meta,
		"",
		0,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling unshield ptokens err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

func getUnshieldStatus(txHash, url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"UnshieldID": txHash,
	}
	params := []interface{}{
		meta,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

func getPortalV4State(beaconHeight int, url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"BeaconHeight": strconv.Itoa(beaconHeight),
	}
	params := []interface{}{
		meta,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

func getSignedRawTxByBatchID(batchID, url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"BatchID": batchID,
	}
	params := []interface{}{
		meta,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

func getBlockchainInfo(url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{}
	params := []interface{}{
		meta,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

func submitExternalTx(incPrivKeyStr, proof, tokenID, batchID, url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"UnshieldProof": proof,
		"PortalTokenID": tokenID,
		"BatchID":       batchID,
	}
	params := []interface{}{
		incPrivKeyStr,
		nil,
		-1,
		-1,
		meta,
		"",
		0,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

func getSubmitConfirmedStatus(txHash, url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"ReqTxID": txHash,
	}
	params := []interface{}{
		meta,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

// createandsendtxwithportalreplacebyfee
func replaceByFeeRequest(incPrivKeyStr, batchID, tokenID, replacementFee, url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"PortalTokenID":  tokenID,
		"BatchID":        batchID,
		"ReplacementFee": replacementFee,
	}
	params := []interface{}{
		incPrivKeyStr,
		nil,
		-1,
		-1,
		meta,
		"",
		0,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

// getportalreplacebyfeestatus
func getReplaceByFeeRequestStatus(txHash, url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"ReqTxID": txHash,
	}
	params := []interface{}{
		meta,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

// getporalsignedrawreplacefeetransaction
func getRequestSigedRawReplaceByFeeTxStatus(txHash, url, rpcMethod string) (map[string]interface{}, error) {
	rpcClient := NewRPCClient()
	meta := map[string]interface{}{
		"TxID": txHash,
	}
	params := []interface{}{
		meta,
	}
	var res ResponseInc
	err := rpcClient.RPCCall(
		"",
		url,
		"",
		rpcMethod,
		params,
		&res,
	)
	if err != nil {
		fmt.Println("calling get shield status err: ", err)
		return nil, err
	}

	if res.RPCError != nil {
		return nil, errors.New(res.RPCError.Message)
	}
	return res.Result.(map[string]interface{}), nil
}

func TestPortalSuiteV4Base(t *testing.T) {
	suite.Run(t, new(PortalV4BaseTestSuite))
}
