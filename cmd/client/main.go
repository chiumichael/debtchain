package main

// this package represents the client that a bank 

import (
	// "crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"debtchain/internal/envelope"
	"debtchain/internal/wallet"
	"debtchain/pkg/utxi"
)

func main() {
	// load keys
	bankwallet, err := wallet.NewWallet("../../keys/bankseed.dat")
	if err != nil {
		fmt.Println("error in opening bank wallet")
		return
	}
	bankAddress, bankAddressString := bankwallet.NewPublicKey()
	// fmt.Println("bankAddressString: ", bankAddressString)

	if (len(bankAddress) > 0) {
	}
	if (len(bankAddressString) > 0) {
	}

	// create the outputs
	clientWallet, err := wallet.NewWallet("../../keys/testseed.dat")
	if err != nil {
		fmt.Println("Error in opening client wallet")
		return
	}
	
	clientAddress, clientAddressString := clientWallet.NewPublicKey()

	debtTx := bankwallet.ConstructDebtTransaction(clientAddress, 50)

	debtTxbytes, _ := json.Marshal(debtTx)
	debtTxbytesbase64 := base64.RawURLEncoding.EncodeToString(debtTxbytes)
	// debtTxbytesbase64 = fmt.Sprintf("\"%v\"",debtTxbytesbase64)

	fmt.Println("debtTxbytesbase64: ", debtTxbytesbase64)

	cmd := envelope.Command{
		Command: 		"IssueDebt",
		Address: 		clientAddressString,
		Transaction: 	debtTxbytesbase64,
	};

	cmdbytes, _ := json.Marshal(cmd)
	cmdbase64 := base64.RawURLEncoding.EncodeToString(cmdbytes)

	// fmt.Println("debtTxbytesbase64: ", debtTxbytesbase64)
	fmt.Println("cmdbase64: ", cmdbase64)

	bodyString := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":\"anything\",\"method\":\"broadcast_tx_commit\",\"params\": {\"tx\": \"%v\"}}",cmdbase64)
	body := strings.NewReader(bodyString)

	req, err := http.NewRequest("POST", "http://localhost:26657", body)
	if err != nil {
		// handle err
	}
	req.Header.Set("Content-Type", "text/plain;")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()	
	
	resBytes, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("resbytes: ", string(resBytes))

	// for illustrative purposes we construct transactions from both a reverse mortgage issuer's 
	// perspetive and from a user perspective. 
	// in a production setting, there would be multiple clients connecting to the blockchain backend
	odtx := utxi.MakeOutstandingDebtTx(debtTx)
	repaymentTx := clientWallet.ConstructRepaymentTransaction(bankAddress,25,odtx,0)
	repaymentTxbytes, _ := json.Marshal(repaymentTx)
	repaymentTxbytes64 := base64.RawURLEncoding.EncodeToString(repaymentTxbytes)

	fmt.Println("debtTx: ", debtTx)
	fmt.Println("\n")
	// print debtTx's output address
	fmt.Println("repaymentTx: ", repaymentTx)

	bwb, bws := bankwallet.PublicKey(1)

	fmt.Println("first bank Address: ", base64.URLEncoding.EncodeToString(bwb))

	// construct output	
	repaymentCmd := envelope.Command{
		Command: "Repayment",
		Address: bws,
		Transaction: repaymentTxbytes64,
	};

	rcmbytes, _ := json.Marshal(repaymentCmd)
	rcmbytes64 := base64.RawURLEncoding.EncodeToString(rcmbytes)

	fmt.Println("rcmbytes64: ", rcmbytes64)

	bodyString2 := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":\"anything\",\"method\":\"broadcast_tx_commit\",\"params\": {\"tx\": \"%v\"}}",rcmbytes64)
	body2 := strings.NewReader(bodyString2)

	req2, err2 := http.NewRequest("POST", "http://localhost:26657", body2)
	if err2 != nil {
		// handle err
	}
	req2.Header.Set("Content-Type", "text/plain;")

	resp2, err2 := http.DefaultClient.Do(req2)
	if err2 != nil {
		// handle err
	}
	defer resp2.Body.Close()	

	resBytes, _ = ioutil.ReadAll(resp2.Body)
	fmt.Println("resbytes2: ", string(resBytes))
}

// the http get request is another way to connect to tendermint
// curl -s 'localhost:26657/abci_query?path=%2Ftest&data="abcd"'