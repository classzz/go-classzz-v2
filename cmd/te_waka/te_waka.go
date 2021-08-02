package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	classzz "github.com/classzz/go-classzz-v2"
	"github.com/classzz/go-classzz-v2/accounts/abi"
	"github.com/classzz/go-classzz-v2/accounts/keystore"
	"github.com/classzz/go-classzz-v2/cmd/utils"
	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/console/prompt"
	"github.com/classzz/go-classzz-v2/core/types"
	"github.com/classzz/go-classzz-v2/core/vm"
	"github.com/classzz/go-classzz-v2/crypto"
	"github.com/classzz/go-classzz-v2/czzclient"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	key   string
	store string
	ip    string
	port  int
)

var (
	abiStaking, _ = abi.JSON(strings.NewReader(vm.TeWakaABI))
	priKey        *ecdsa.PrivateKey
	from          common.Address
	czzValue      uint64
	holder        common.Address
)

const (
	datadirPrivateKey      = "key"
	datadirDefaultKeyStore = "keystore"
	TeWakaAmount           = 100
)

func checkFee(fee *big.Int) {
	if fee.Sign() < 0 || fee.Cmp(vm.Base) > 0 {
		printError("Please set correct fee value")
	}
}

func getPubKey(ctx *cli.Context, conn *czzclient.Client) (string, []byte, error) {
	var (
		pubkey string
		err    error
	)

	if ctx.GlobalIsSet(PubKeyKeyFlag.Name) {
		pubkey = ctx.GlobalString(PubKeyKeyFlag.Name)
	} else if ctx.GlobalIsSet(BFTKeyKeyFlag.Name) {
		bftKey, err := crypto.HexToECDSA(ctx.GlobalString(BFTKeyKeyFlag.Name))
		if err != nil {
			printError("bft key error", err)
		}
		pk := crypto.FromECDSAPub(&bftKey.PublicKey)
		pubkey = common.Bytes2Hex(pk)
	} else {
		pubkey, err = conn.Pubkey(context.Background())
		if err != nil {
			printError("get pubkey error", err)
		}
	}

	pk := common.Hex2Bytes(pubkey)
	if err = vm.ValidPk(pk); err != nil {
		printError("ValidPk error", err)
	}
	return pubkey, pk, err
}

func sendContractTransaction(client *czzclient.Client, from, toAddress common.Address, value *big.Int, privateKey *ecdsa.PrivateKey, input []byte) common.Hash {
	// Ensure a valid value field and resolve the account nonce
	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	gasLimit := uint64(2100000) // in units
	// If the contract surely has code (or code is not needed), estimate the transaction
	//msg := classzz.CallMsg{From: from, To: &toAddress, GasPrice: gasPrice, Value: value, Data: input}
	//gasLimit, err = client.EstimateGas(context.Background(), msg)
	//if err != nil {
	//	fmt.Println("Contract exec failed", err)
	//}
	//if gasLimit < 1 {
	//	gasLimit = 866328
	//}

	// Create the transaction, sign it and schedule it for execution
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, input)

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("TX data nonce ", nonce, " transfer value ", value, " gasLimit ", gasLimit, " gasPrice ", gasPrice, " chainID ", chainID)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	return signedTx.Hash()
}

func createKs() {
	ks := keystore.NewKeyStore("./createKs", keystore.StandardScryptN, keystore.StandardScryptP)
	password := "secret"
	account, err := ks.NewAccount(password)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(account.Address.Hex()) // 0x20F8D42FB0F667F2E53930fed426f225752453b3
}

func importKs(password string) common.Address {
	file, err := getAllFile(datadirDefaultKeyStore)
	if err != nil {
		log.Fatal(err)
	}
	cks, _ := filepath.Abs(datadirDefaultKeyStore)

	jsonBytes, err := ioutil.ReadFile(filepath.Join(cks, file))
	if err != nil {
		log.Fatal(err)
	}

	//password := "secret"
	key, err := keystore.DecryptKey(jsonBytes, password)
	if err != nil {
		log.Fatal(err)
	}
	priKey = key.PrivateKey
	from = crypto.PubkeyToAddress(priKey.PublicKey)

	fmt.Println("address ", from.Hex())
	return from
}

func loadPrivateKey(path string) common.Address {
	var err error
	if path == "" {
		file, err := getAllFile(datadirPrivateKey)
		if err != nil {
			printError(" getAllFile file name error", err)
		}
		kab, _ := filepath.Abs(datadirPrivateKey)
		path = filepath.Join(kab, file)
	}
	priKey, err = crypto.LoadECDSA(path)
	if err != nil {
		printError("LoadECDSA error", err)
	}
	from = crypto.PubkeyToAddress(priKey.PublicKey)
	return from
}

func getAllFile(path string) (string, error) {
	rd, err := ioutil.ReadDir(path)
	if err != nil {
		printError("path ", err)
	}
	for _, fi := range rd {
		if fi.IsDir() {
			fmt.Printf("[%s]\n", path+"\\"+fi.Name())
			getAllFile(path + fi.Name() + "\\")
			return "", errors.New("path error")
		} else {
			fmt.Println(path, "dir has ", fi.Name(), "file")
			return fi.Name(), nil
		}
	}
	return "", err
}

func printError(error ...interface{}) {
	log.Fatal(error)
}

//func czzToWei(ctx *cli.Context, zero bool) *big.Int {
//	czzValue = ctx.GlobalUint64(CzzValueFlag.Name)
//	if !zero && czzValue <= 0 {
//		printError("Value must bigger than 0")
//	}
//	baseUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
//	value := new(big.Int).Mul(big.NewInt(int64(czzValue)), baseUnit)
//	return value
//}

func czzToWei(amount uint64) *big.Int {
	baseUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	value := new(big.Int).Mul(big.NewInt(200), baseUnit)
	fmt.Println(value.String())
	return value
}

func weiToCzz(value *big.Int) uint64 {
	baseUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	valueT := new(big.Int).Div(value, baseUnit).Uint64()
	return valueT
}

func getResult(conn *czzclient.Client, txHash common.Hash, contract bool, delegate bool) {
	fmt.Println("Please waiting ", " txHash ", txHash.String())

	count := 0
	for {
		time.Sleep(time.Millisecond * 200)
		_, isPending, err := conn.TransactionByHash(context.Background(), txHash)
		if err != nil {
			log.Fatal(err)
		}
		count++
		if !isPending {
			break
		}
		if count >= 40 {
			fmt.Println("Please use querytx sub command query later.")
			os.Exit(0)
		}
	}

	queryTx(conn, txHash, contract, false, delegate)
}

func queryTx(conn *czzclient.Client, txHash common.Hash, contract bool, pending bool, delegate bool) {

	if pending {
		_, isPending, err := conn.TransactionByHash(context.Background(), txHash)
		if err != nil {
			log.Fatal(err)
		}
		if isPending {
			println("In tx_pool no validator  process this, please query later")
			os.Exit(0)
		}
	}

	receipt, err := conn.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		log.Fatal(err)
	}

	if receipt.Status == types.ReceiptStatusSuccessful {
		block, err := conn.BlockByHash(context.Background(), receipt.BlockHash)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Transaction Success", " block Number", receipt.BlockNumber.Uint64(), " block txs", len(block.Transactions()), "blockhash", block.Hash().Hex())
		if contract && common.IsHexAddress(from.Hex()) {
			queryStakingInfo(conn, false, delegate)
		}
	} else if receipt.Status == types.ReceiptStatusFailed {
		fmt.Println("Transaction Failed ", " Block Number", receipt.BlockNumber.Uint64())
	}
}

func packInput(abiMethod string, params ...interface{}) []byte {
	input, err := abiStaking.Pack(abiMethod, params...)
	if err != nil {
		printError(abiMethod, " error ", err)
	}
	return input
}

func PrintBalance(conn *czzclient.Client, from, toaddress common.Address) {
	balance, err := conn.BalanceAt(context.Background(), from, nil)
	if err != nil {
		log.Fatal(err)
	}
	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	czzValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	sbalance, err := conn.BalanceAt(context.Background(), toaddress, nil)
	fmt.Println("Your wallet valid balance is ", czzValue, "czz", " lock balance is ", common.ToCzz(sbalance), "czz ")
}

func loadPrivate(ctx *cli.Context) {
	key = ctx.GlobalString(KeyFlag.Name)
	store = ctx.GlobalString(KeyStoreFlag.Name)
	if key != "" {
		loadPrivateKey(key)
	} else if store != "" {
		loadSigningKey(store)
	} else {
		printError("Must specify --key or --keystore")
	}

	if priKey == nil {
		printError("load privateKey failed")
	}
}

func dialConn(ctx *cli.Context) (*czzclient.Client, string) {
	ip = ctx.GlobalString(utils.LegacyRPCListenAddrFlag.Name)
	port = ctx.GlobalInt(utils.LegacyRPCPortFlag.Name)

	url := fmt.Sprintf("http://%s", fmt.Sprintf("%s:%d", ip, port))
	// Create an IPC based RPC connection to a remote node
	conn, err := czzclient.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to the Truechain client: %v", err)
	}
	return conn, url
}

func printBaseInfo(conn *czzclient.Client, url string) *types.Header {
	header, err := conn.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	if common.IsHexAddress(from.Hex()) {
		fmt.Println("Connect url ", url, " current number ", header.Number.String(), " address ", from.Hex())
	} else {
		fmt.Println("Connect url ", url, " current number ", header.Number.String())
	}

	return header
}

// loadSigningKey loads a private key in Ethereum keystore format.
func loadSigningKey(keyfile string) common.Address {
	keyjson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		printError(fmt.Errorf("failed to read the keyfile at '%s': %v", keyfile, err))
	}
	password, _ := prompt.Stdin.PromptPassword("Please enter the password for '" + keyfile + "': ")
	//password := "secret"
	key, err := keystore.DecryptKey(keyjson, password)
	if err != nil {
		printError(fmt.Errorf("error decrypting key: %v", err))
	}
	priKey = key.PrivateKey
	from = crypto.PubkeyToAddress(priKey.PublicKey)
	//fmt.Println("address ", from.Hex(), "key", hex.EncodeToString(crypto.FromECDSA(priKey)))
	return from
}

//func queryRewardInfo(conn *czzclient.Client, number uint64, start bool) {
//	sheader, err := conn.SnailHeaderByNumber(context.Background(), nil)
//	if err != nil {
//		printError("get snail block error", err)
//	}
//	queryReward := uint64(0)
//	currentReward := sheader.Number.ToInt().Uint64() - SnailRewardInterval
//	if number > currentReward {
//		printError("reward no release current reward height ", currentReward)
//	} else if number > 0 || start {
//		queryReward = number
//	} else {
//		queryReward = currentReward
//	}
//	var crc map[string]interface{}
//	crc, err = conn.GetChainRewardContent(context.Background(), from, new(big.Int).SetUint64(queryReward))
//	if err != nil {
//		printError("get chain reward content error", err)
//	}
//	if info, ok := crc["stakingReward"]; ok {
//		if info, ok := info.([]interface{}); ok {
//			fmt.Println("queryRewardInfo", info)
//		}
//	}
//}

func queryStakingInfo(conn *czzclient.Client, query bool, delegate bool) {
	header, err := conn.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	var input []byte
	if delegate {
		input = packInput("getDelegate", from, holder)
	} else {
		input = packInput("getDeposit", from)
	}
	msg := classzz.CallMsg{From: from, To: &vm.TeWaKaAddress, Data: input}
	output, err := conn.CallContract(context.Background(), msg, header.Number)
	if err != nil {
		printError("method CallContract error", err)
	}
	if len(output) != 0 {

		inter, err := abiStaking.Unpack("getDeposit", output)
		if err != nil {
			printError("abi error", err)
		}
		args, ok := inter[0].(struct {
			Staked   *big.Int
			Locked   *big.Int
			Unlocked *big.Int
		})
		if !ok {
			fmt.Println("failed")
			return
		}

		if err != nil {
			printError("abi error", err)
		}
		fmt.Println("Staked ", args.Staked.String(), "wei =", common.ToCzz(args.Staked), "true Locked ",
			args.Locked.String(), " wei =", common.ToCzz(args.Locked), "true",
			"Unlocked ", args.Unlocked.String(), " wei =", common.ToCzz(args.Unlocked), "true")
		if query && args.Locked.Sign() > 0 {
			//lockAssets, err := conn.GetLockedAsset(context.Background(), from, header.Number)
			//if err != nil {
			//	printError("GetLockedAsset error", err)
			//}
			//for k, v := range lockAssets {
			//	for m, n := range v.LockValue {
			//		if !n.Locked {
			//			fmt.Println("Your can instant withdraw", " count value ", n.Amount, " true")
			//		} else {
			//			if n.EpochID > 0 || n.Amount != "0" {
			//				fmt.Println("Your can withdraw after height", n.Height.Uint64(), " count value ", n.Amount, " true  index", k+m, " lock ", n.Locked)
			//			}
			//		}
			//	}
			//}
		}
	} else {
		fmt.Println("Contract query failed result len == 0")
	}
}
