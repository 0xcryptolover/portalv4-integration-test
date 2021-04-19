package awesomeProject1

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including assertion methods.
type PortalIntegrationTestSuite struct {
	*PortalV4BaseTestSuite

	Host             string
	Username         string
	Password         string
	BTCClient        *rpcclient.Client
	BTCTempAddress   btcutil.Address
	BTCTestAddress   btcutil.Address
	BTCTokenID       string
	MultiSig         string
	TestAmount       uint64
	ExternalAddress  string
	UnshieldAmount   uint64
	MasterPubs       [][]byte
	NumSigsRequired  int
	ReplaceFeeAmount string
}

type BatchIDProof struct {
	BatchID   string
	Proof     string
	Unshields []interface{}
}

func NewPortalIntegrationTestSuite(tradingTestSuite *PortalV4BaseTestSuite) *PortalIntegrationTestSuite {
	return &PortalIntegrationTestSuite{
		PortalV4BaseTestSuite: tradingTestSuite,
	}
}

// Make sure that VariableThatShouldStartAtFive is set to five
// before each test
func (pg *PortalIntegrationTestSuite) SetupSuite() {
	fmt.Println("Setting up the suite...")

	var err error
	//remove container if already running
	exec.Command("/bin/sh", "-c", "docker rm -f regtest").Output()
	exec.Command("/bin/sh", "-c", "docker rm -f portalv4").Output()
	//setup bitcoin regtest
	_, err = exec.Command("/bin/sh", "-c", "docker run -d --name regtest -p 18443:18443 -p 18444:18444 ruimarinho/bitcoin-core -printtoconsole -txindex=1 -reindex-chainstate=1 -regtest=1 -rpcallowip=172.17.0.0/16 -rpcbind=0.0.0.0 -rpcport=18443 -port=18444 -server=1 -rpcuser=thach -rpcpassword=deptrai -fallbackfee=1 -maxtxfee=1000000 -bytespersigop=1000 -datacarriersize=1000").Output()
	require.Equal(pg.T(), nil, err)
	time.Sleep(3 * time.Second)
	pg.Host = "127.0.0.1:18443"
	pg.Username = "thach"
	pg.Password = "deptrai"
	pg.TestAmount = 1e6
	pg.BTCTokenID = "ef5947f70ead81a76a53c7c8b7317dd5245510c665d3a13921dc9a581188728b"
	pg.ExternalAddress = "bcrt1q8y058dhaeshdzxfh7jem2zmj3kk05gj5e8nw7e"
	pg.ReplaceFeeAmount = "51000"
	pg.UnshieldAmount = 5e6
	pg.NumSigsRequired = 3
	pg.MasterPubs = [][]byte{
		{0x3, 0xb2, 0xd3, 0x16, 0x7d, 0x94, 0x9c, 0x25, 0x3, 0xe6, 0x9c, 0x9f, 0x29, 0x78, 0x7d, 0x9c, 0x8, 0x8d, 0x39, 0x17, 0x8d, 0xb4, 0x75, 0x40, 0x35, 0xf5, 0xae, 0x6a, 0xf0, 0x17, 0x12, 0x11, 0x0},
		{0x3, 0x98, 0x7a, 0x87, 0xd1, 0x99, 0x13, 0xbd, 0xe3, 0xef, 0xf0, 0x55, 0x79, 0x2, 0xb4, 0x90, 0x57, 0xed, 0x1c, 0x9c, 0x8b, 0x32, 0xf9, 0x2, 0xbb, 0xbb, 0x85, 0x71, 0x3a, 0x99, 0x1f, 0xdc, 0x41},
		{0x3, 0x73, 0x23, 0x5e, 0xb1, 0xc8, 0xf1, 0x84, 0xe7, 0x59, 0x17, 0x6c, 0xe3, 0x87, 0x37, 0xb7, 0x91, 0x19, 0x47, 0x1b, 0xba, 0x63, 0x56, 0xbc, 0xab, 0x8d, 0xcc, 0x14, 0x4b, 0x42, 0x99, 0x86, 0x1},
		{0x3, 0x29, 0xe7, 0x59, 0x31, 0x89, 0xca, 0x7a, 0xf6, 0x1, 0xb6, 0x35, 0x67, 0x3d, 0xb1, 0x53, 0xd4, 0x19, 0xd7, 0x6, 0x19, 0x3, 0x2a, 0x32, 0x94, 0x57, 0x76, 0xb2, 0xb3, 0x80, 0x65, 0xe1, 0x5d},
	}
	initWallet := fmt.Sprintf("curl -u %v:%v --data-binary '{\"jsonrpc\": \"1.0\", \"id\":\"curltest\", \"method\": \"createwallet\", \"params\": [\"testwallet\"] }' -H 'content-type: text/plain;' %v/", pg.Username, pg.Password, pg.Host)
	_, err = exec.Command("/bin/sh", "-c", initWallet).Output()
	require.Equal(pg.T(), nil, err)
	connCfg := &rpcclient.ConnConfig{
		Host:         pg.Host,
		Endpoint:     "http",
		User:         pg.Username,
		Pass:         pg.Password,
		DisableTLS:   true,
		HTTPPostMode: true,
	}
	pg.BTCClient, err = rpcclient.New(connCfg, nil)
	require.Equal(pg.T(), nil, err)
	pg.BTCTempAddress, err = pg.BTCClient.GetNewAddress("temp")
	require.Equal(pg.T(), nil, err)
	fmt.Printf("bitcoin temp address: %v \n", pg.BTCTempAddress.String())
	pg.BTCTestAddress, err = pg.BTCClient.GetNewAddress("test")
	require.Equal(pg.T(), nil, err)
	fmt.Printf("bitcoin test address: %v \n", pg.BTCTestAddress.String())
	_, err = pg.BTCClient.GenerateToAddress(200, pg.BTCTempAddress, nil)
	require.Equal(pg.T(), nil, err)
	// send to test address 10 bitcoin
	_, err = pg.BTCClient.SendToAddress(pg.BTCTestAddress, btcutil.Amount(1e11))
	require.Equal(pg.T(), nil, err)
	// gen 7 blocks
	_, err = pg.BTCClient.GenerateToAddress(7, pg.BTCTempAddress, nil)
	require.Equal(pg.T(), nil, err)
	// import multisig address
	watchAddress := fmt.Sprintf("curl -u %v:%v --data-binary '{\"jsonrpc\": \"1.0\", \"id\":\"curltest\", \"method\": \"importaddress\", \"params\": [\"2NGFTTKNj59NGmjQpajsEXGxwf9SP8gvJiv\"] }' -H 'content-type: text/plain;' %v/", pg.Username, pg.Password, pg.Host)
	_, err = exec.Command("/bin/sh", "-c", watchAddress).Output()
	require.Equal(pg.T(), nil, err)

	_, err = exec.Command("/bin/sh", "-c", "docker run -d --name portalv4 -p 9334:9334 -p 9338:9338 portalv4").Output()
	require.Equal(pg.T(), nil, err)
	time.Sleep(10 * time.Second)

	for {
		fmt.Println("Calling to incognito to fullnode to check is it ready, please wait...")
		time.Sleep(5 * time.Second)
		if checkRepsonse(pg.IncBridgeHost) {
			break
		}
	}

	// wait until beacon block reach out to 10 height
	time.Sleep(120 * time.Second)
}

func (pg *PortalIntegrationTestSuite) TearDownSuite() {
	fmt.Println("Tearing down the suite...")
	var err error
	_, err = exec.Command("/bin/sh", "-c", "docker rm -f portalv4").Output()
	require.Equal(pg.T(), nil, err)
	_, err = exec.Command("/bin/sh", "-c", "docker rm -f regtest").Output()
	require.Equal(pg.T(), nil, err)
	pg.BTCClient.Shutdown()
}

func (pg *PortalIntegrationTestSuite) SetupTest() {
	fmt.Println("Setting up the test...")
}

func (pg *PortalIntegrationTestSuite) TearDownTest() {
	fmt.Println("Tearing down the test...")
}

func (pg *PortalIntegrationTestSuite) Test1Shield() {
	fmt.Println("============ TEST CUSTODIAN DEPOSIT ===========")
	fmt.Println("------------ STEP 0: declaration & initialization --------------")
	utxos, err := pg.BTCClient.ListUnspentMinMaxAddresses(6, 9999999, []btcutil.Address{pg.BTCTestAddress})
	require.Equal(pg.T(), nil, err)

	inputs := make([]btcjson.TransactionInput, len(utxos))
	for i, v := range utxos {
		inputs[i] = btcjson.TransactionInput{Vout: v.Vout, Txid: v.TxID}
	}
	_, pg.MultiSig, err = GenerateOTMultisigAddress(&chaincfg.RegressionNetParams, pg.MasterPubs, pg.NumSigsRequired, pg.IncPaymentAddrStr)
	require.Equal(pg.T(), nil, err)
	err = pg.BTCClient.ImportAddress(pg.MultiSig)
	require.Equal(pg.T(), nil, err)
	outputs := make([]map[string]interface{}, 2)
	outputs[0] = make(map[string]interface{}, 0)
	outputs[0][pg.MultiSig] = float64(pg.TestAmount) / BTC_DECIMAL
	outputs[1] = make(map[string]interface{}, 0)
	outputs[1][pg.BTCTestAddress.String()] = float64(99998000000) / BTC_DECIMAL

	fmt.Println("------------ Convert To PrivacyV2 --------------")
	result, err := submitKey(pg.IncOTAPriKey, pg.IncBridgeHost, "submitkey")
	fmt.Println(result)
	time.Sleep(40 * time.Second)
	result, err = convertToPrivacyV2(pg.IncPrivKeyStr, pg.IncBridgeHost, "createconvertcoinver1tover2transaction")
	fmt.Println(result)
	time.Sleep(40 * time.Second)

	fmt.Println("------------ STEP 1: Shield Bitcoin --------------")
	// send bitcoin to incognito multisig address
	sendBTCTxID, err := createAndSendRawTx(pg.BTCClient, pg.Host, pg.Username+":"+pg.Password, pg.BTCTestAddress, inputs, outputs)
	require.Equal(pg.T(), nil, err)
	fmt.Printf("bitcoin tx id: %v \n", sendBTCTxID)
	fmt.Printf("send amount: %v btc\n", float64(pg.TestAmount)/1e8)
	fmt.Printf("index: 0 \n")
	fmt.Printf("payment address: %v \n", pg.IncPaymentAddrStr)
	pg.BTCClient.GenerateToAddress(7, pg.BTCTempAddress, nil)
	btcProof := buildProof(pg.BTCClient, sendBTCTxID)
	require.NotEqual(pg.T(), "", btcProof)
	fmt.Printf("btcproof: %v \n", btcProof)

	// call shield ptoken
	shieldResult, err := shieldToken("createandsendtxshieldingrequest", pg.IncBridgeHost, btcProof, pg.IncPrivKeyStr, pg.IncPaymentAddrStr, pg.BTCTokenID)
	fmt.Printf("Shield result: %v \n", shieldResult)
	require.Equal(pg.T(), nil, err)
	shieldTxID := shieldResult["TxID"].(string)
	for {
		time.Sleep(5 * time.Second)
		shieldStatus, _ := getShieldStatus(shieldTxID, pg.IncBridgeHost, "getportalshieldingrequeststatus")
		if shieldStatus != nil {
			fmt.Printf("shield status: %v \n", shieldStatus)
			require.Equal(pg.T(), float64(1), shieldStatus["Status"].(float64))
			break
		}
	}
	// wait for minting token
	time.Sleep(40 * time.Second)
	fmt.Println("------------ STEP 2: Request unshield Shield Bitcoin --------------")
	unshieldRes, err := unshieldPToken("createandsendtxwithportalv4unshieldrequest", pg.IncBridgeHost, pg.IncPrivKeyStr, pg.IncPaymentAddrStr, pg.ExternalAddress, pg.BTCTokenID, strconv.FormatUint(pg.UnshieldAmount, 10))
	require.Equal(pg.T(), nil, err)
	fmt.Printf("Unshield result: %v \n", unshieldRes)
	unshieldTxID := unshieldRes["TxID"].(string)
	// expect unshield pending
	for {
		time.Sleep(5 * time.Second)
		unshieldStatus, _ := getUnshieldStatus(unshieldTxID, pg.IncBridgeHost, "getportalunshieldrequeststatus")
		if unshieldStatus != nil {
			fmt.Printf("unshield status: %v \n", unshieldStatus)
			require.Equal(pg.T(), float64(0), unshieldStatus["Status"].(float64))
			break
		}
	}

	//wait until batchid created
	for {
		time.Sleep(5 * time.Second)
		unshieldStatus, _ := getUnshieldStatus(unshieldTxID, pg.IncBridgeHost, "getportalunshieldrequeststatus")
		if unshieldStatus["Status"].(float64) == 1 {
			break
		}
	}

	portalUnshieldRequests := pg.GetUnshileBatchs()
	submitConfirmedProof := make([]*BatchIDProof, 0)
	for _, v := range portalUnshieldRequests {
		batchID := v.(map[string]interface{})["BatchID"].(string)
		for {
			time.Sleep(5 * time.Second)
			resignedRawTxResult, err := getSignedRawTxByBatchID(batchID, pg.IncBridgeHost, "getportalsignedrawtransaction")
			if err != nil {
				continue
			}
			batchIDProof := pg.BroadcastRawTx(batchID, resignedRawTxResult["SignedTx"].(string))
			unshields := v.(map[string]interface{})["UnshieldsID"].([]interface{})
			batchIDProof.Unshields = unshields
			submitConfirmedProof = append(submitConfirmedProof, batchIDProof)
			break
		}
	}

	fmt.Println("------------ STEP 3: Submit confirmed external tx --------------")
	for _, proof := range submitConfirmedProof {
		result, err := submitExternalTx(pg.IncPrivKeyStr, proof.Proof, pg.BTCTokenID, proof.BatchID, pg.IncBridgeHost, "createandsendtxwithportalsubmitconfirmedtx")
		require.Equal(pg.T(), nil, err)
		//wait until submitConfirmedProof return status
		for {
			time.Sleep(5 * time.Second)
			submitConfirmedStatus, _ := getSubmitConfirmedStatus(result["TxID"].(string), pg.IncBridgeHost, "getportalsubmitconfirmedtxstatus")
			if submitConfirmedStatus != nil {
				require.Equal(pg.T(), float64(1), submitConfirmedStatus["Status"].(float64))
				break
			}
		}
		for _, v := range proof.Unshields {
			unshieldCompleted, _ := getUnshieldStatus(v.(string), pg.IncBridgeHost, "getportalunshieldrequeststatus")
			require.Equal(pg.T(), float64(2), unshieldCompleted["Status"].(float64))
		}
	}

	fmt.Println("------------ STEP 2': Testcase Request replace by fee --------------")
	portalUnshieldRequests = pg.GetUnshileBatchs()
	submitConfirmedProof = make([]*BatchIDProof, 0)
	// init fee 50000 pbtc ~ 5000 satoshi
	unshieldRes, err = unshieldPToken("createandsendtxwithportalv4unshieldrequest", pg.IncBridgeHost, pg.IncPrivKeyStr, pg.IncPaymentAddrStr, pg.ExternalAddress, pg.BTCTokenID, strconv.FormatUint(pg.UnshieldAmount, 10))
	require.Equal(pg.T(), nil, err)
	fmt.Printf("Unshield result: %v \n", unshieldRes)
	unshieldTxID = unshieldRes["TxID"].(string)
	// expect unshield pending
	for {
		time.Sleep(5 * time.Second)
		unshieldStatus, _ := getUnshieldStatus(unshieldTxID, pg.IncBridgeHost, "getportalunshieldrequeststatus")
		if unshieldStatus != nil {
			fmt.Printf("unshield status: %v \n", unshieldStatus)
			require.Equal(pg.T(), float64(0), unshieldStatus["Status"].(float64))
			break
		}
	}

	//wait until batchid created
	for {
		time.Sleep(5 * time.Second)
		unshieldStatus, _ := getUnshieldStatus(unshieldTxID, pg.IncBridgeHost, "getportalunshieldrequeststatus")
		if unshieldStatus["Status"].(float64) == 1 {
			break
		}
	}
	time.Sleep(130 * time.Second)
	for _, v := range portalUnshieldRequests {
		batchID := v.(map[string]interface{})["BatchID"].(string)
		replaceByFee, err := replaceByFeeRequest(pg.IncPrivKeyStr, batchID, pg.BTCTokenID, pg.ReplaceFeeAmount, pg.IncBridgeHost, "createandsendtxwithportalreplacebyfee")
		require.Equal(pg.T(), nil, err)
		for {
			time.Sleep(5 * time.Second)
			replaceByFeeStatus, _ := getReplaceByFeeRequestStatus(replaceByFee["TxID"].(string), pg.IncBridgeHost, "getportalreplacebyfeestatus")
			if replaceByFeeStatus != nil {
				fmt.Printf("replace by fee status: %v \n", replaceByFeeStatus)
				require.Equal(pg.T(), float64(1), replaceByFeeStatus["Status"].(float64))
				break
			}
		}

		for {
			time.Sleep(5 * time.Second)
			resignedRawTxResult, err := getRequestSigedRawReplaceByFeeTxStatus(replaceByFee["TxID"].(string), pg.IncBridgeHost, "getporalsignedrawreplacebyfeetransaction")
			if err != nil {
				continue
			}
			batchIDProof := pg.BroadcastRawTx(batchID, resignedRawTxResult["SignedTx"].(string))
			unshields := v.(map[string]interface{})["UnshieldsID"].([]interface{})
			batchIDProof.Unshields = unshields
			submitConfirmedProof = append(submitConfirmedProof, batchIDProof)
			break
		}
	}

	fmt.Println("------------ STEP 3': Submit confirmed replace by fee external tx --------------")
	for _, proof := range submitConfirmedProof {
		result, err := submitExternalTx(pg.IncPrivKeyStr, proof.Proof, pg.BTCTokenID, proof.BatchID, pg.IncBridgeHost, "createandsendtxwithportalsubmitconfirmedtx")
		require.Equal(pg.T(), nil, err)
		//wait until submitConfirmedProof return status
		for {
			time.Sleep(5 * time.Second)
			submitConfirmedStatus, _ := getSubmitConfirmedStatus(result["TxID"].(string), pg.IncBridgeHost, "getportalsubmitconfirmedtxstatus")
			if submitConfirmedStatus != nil {
				require.Equal(pg.T(), float64(1), submitConfirmedStatus["Status"].(float64))
				break
			}
		}

		for _, v := range proof.Unshields {
			unshieldCompleted, _ := getUnshieldStatus(v.(string), pg.IncBridgeHost, "getportalunshieldrequeststatus")
			require.Equal(pg.T(), float64(2), unshieldCompleted["Status"].(float64))
		}
	}
}

func checkRepsonse(url string) bool {
	resp, err := http.Get(url)
	if err != nil || resp == nil {
		fmt.Println("Incognito chain is running please wait...")
		return false
	}
	return true
}

func (pg *PortalIntegrationTestSuite) GetUnshileBatchs() map[string]interface{} {
	blockchainInfo, err := getBlockchainInfo(pg.IncBridgeHost, "getblockchaininfo")
	require.Equal(pg.T(), nil, err)
	beaconInfo := blockchainInfo["BestBlocks"].(map[string]interface{})["-1"].(map[string]interface{})
	beaconHeight := beaconInfo["Height"].(float64)
	portalState, err := getPortalV4State(int(beaconHeight), pg.IncBridgeHost, "getportalv4state")
	require.Equal(pg.T(), nil, err)
	portalProccessedUnshield := portalState["ProcessedUnshieldRequests"].(map[string]interface{})[pg.BTCTokenID]
	return portalProccessedUnshield.(map[string]interface{})
}

func (pg *PortalIntegrationTestSuite) BroadcastRawTx(batchID, rawTx string) *BatchIDProof {
	hexRawTx, err := hex.DecodeString(rawTx)
	require.Equal(pg.T(), nil, err)
	buffer := bytes.NewReader(hexRawTx)
	redeemTx := wire.NewMsgTx(wire.TxVersion)
	err = redeemTx.Deserialize(buffer)
	btcTx, err := pg.BTCClient.SendRawTransaction(redeemTx, true)
	require.Equal(pg.T(), nil, err)
	fmt.Printf("btc tx id: %v \n", btcTx.String())
	pg.BTCClient.GenerateToAddress(7, pg.BTCTempAddress, nil)
	return &BatchIDProof{BatchID: batchID, Proof: buildProof(pg.BTCClient, btcTx)}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestPortalIntegrationTestSuite(t *testing.T) {
	fmt.Println("Starting entry point for portalv4 test suite...")

	pg := new(PortalV4BaseTestSuite)
	suite.Run(t, pg)

	portalv4Suite := NewPortalIntegrationTestSuite(pg)
	suite.Run(t, portalv4Suite)

	fmt.Println("Finishing entry point for 0x test suite...")
}

func GenerateOTMultisigAddress(chainParams *chaincfg.Params, masterPubKeys [][]byte, numSigsRequired int, publicSeed string) ([]byte, string, error) {
	if len(masterPubKeys) < numSigsRequired || numSigsRequired < 0 {
		return []byte{}, "", fmt.Errorf("Invalid signature requirment")
	}

	pubKeys := [][]byte{}
	// this Incognito address is marked for the address that received change UTXOs
	if publicSeed == "" {
		pubKeys = masterPubKeys[:]
	} else {
		chainCode := chainhash.HashB([]byte(publicSeed))
		for idx, masterPubKey := range masterPubKeys {
			// generate BTC child public key for this Incognito address
			extendedBTCPublicKey := hdkeychain.NewExtendedKey(chainParams.HDPublicKeyID[:], masterPubKey, chainCode, []byte{}, 0, 0, false)
			extendedBTCChildPubKey, _ := extendedBTCPublicKey.Child(0)
			childPubKey, err := extendedBTCChildPubKey.ECPubKey()
			if err != nil {
				return []byte{}, "", fmt.Errorf("Master BTC Public Key #%v: %v is invalid", idx, masterPubKey)
			}
			pubKeys = append(pubKeys, childPubKey.SerializeCompressed())
		}
	}

	// create redeem script for m of n multi-sig
	builder := txscript.NewScriptBuilder()
	// add the minimum number of needed signatures
	builder.AddOp(byte(txscript.OP_1 - 1 + numSigsRequired))
	// add the public key to redeem script
	for _, pubKey := range pubKeys {
		builder.AddData(pubKey)
	}
	// add the total number of public keys in the multi-sig script
	builder.AddOp(byte(txscript.OP_1 - 1 + len(pubKeys)))
	// add the check-multi-sig op-code
	builder.AddOp(txscript.OP_CHECKMULTISIG)

	redeemScript, err := builder.Script()
	if err != nil {
		return []byte{}, "", err
	}

	scriptHash := sha256.Sum256(redeemScript)
	multi, err := btcutil.NewAddressWitnessScriptHash(scriptHash[:], chainParams)
	if err != nil {
		return []byte{}, "", err
	}

	return redeemScript, multi.EncodeAddress(), nil
}
