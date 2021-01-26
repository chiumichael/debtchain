package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"io/ioutil"
	"math/big"

	"github.com/FactomProject/btcutilecc"
	"github.com/tyler-smith/go-bip32"

	"debtchain/pkg/utxi"
)

type Wallet struct {
	Seed 		[]uint8
	MasterKey 	*bip32.Key
	mostRecentKey	uint32
}

func (w *Wallet) GetToWork() {
}

func NewWallet(seedpath string) (*Wallet, error) {
	seed, err := ioutil.ReadFile(seedpath)
	if err != nil {
		// fmt.Println("error reading file")
		return nil, errors.New("error reading file")
	}
	masterkey, errm := bip32.NewMasterKey(seed)
	if errm != nil {
		// handle error
	}
	return &Wallet{seed, masterkey, 0}, nil
}

func (w *Wallet) NewPublicKey() ([]byte, string) {
	w.mostRecentKey = w.mostRecentKey + 1
	return w.PublicKey(w.mostRecentKey)
}

func (w *Wallet) PublicKey(which uint32) ([]byte, string) {
	childkey, _ := w.MasterKey.NewChildKey(which)
	childkey_pk := childkey.PublicKey()
	return childkey_pk.Key, childkey_pk.String()
}

func (w *Wallet) signPublicKey(address []byte) utxi.UnLockingScript {

	curve := btcutil.Secp256k1()

	// we use the first child key to sign everything from the perspective of the creditor
	childKey1, _ := w.MasterKey.NewChildKey(0)
	x, y := curve.ScalarBaseMult(childKey1.Key)

	gopublic := ecdsa.PublicKey{curve,x,y}

	var pkInt big.Int
	pkInt.SetBytes(childKey1.Key)
	gopriv := ecdsa.PrivateKey{gopublic, &pkInt}

	r, s, _ := ecdsa.Sign(rand.Reader, &gopriv, address)
	signed := utxi.EcdsaSignature{r,s}

	return utxi.UnLockingScript{address, signed}
}

func (w* Wallet) createDebtInput(debtorAddress []byte) utxi.TxInput {

	var vout int64
	// one way to indicate that this is a debt input 
	vout = -2

	// note that we can choose the publick key to record onto the blockchain
	childKey, _ := w.MasterKey.NewChildKey(1)
	signature := w.signPublicKey(debtorAddress)

	return utxi.TxInput{
		// this allows us to record the originator of the debt 
		Txid: childKey.PublicKey().Key,
		Vout: vout,
		ScriptSig: signature,
	}
}

func (w *Wallet) ConstructDebtTransaction(debtorAddress []byte, amount uint64) utxi.Transaction {

	// construct input
	input := w.createDebtInput(debtorAddress)
	output := utxi.ConstructOutput(debtorAddress, amount)

	return utxi.Transaction{
		[]utxi.TxInput{input}, 
		[]utxi.TxOutput{output},
	}
}

func (w *Wallet) CreatePaymentInput(txId, utxoAddress []byte, vout int64) utxi.TxInput {
	return utxi.TxInput {
		Txid: txId,
		Vout: vout,
		ScriptSig: w.signPublicKey(utxoAddress),
	}
}

func (w *Wallet) ConstructRepaymentTransaction(repaymentAddress []byte, repaymentAmt uint64, utxoTx utxi.Transaction, vout int64) utxi.Transaction {

	input := w.CreatePaymentInput(utxoTx.Hash(), utxoTx.Outputs[vout].RecipientAddr(), vout)
	output := utxi.ConstructOutput(repaymentAddress, repaymentAmt)

	return utxi.Transaction{
		[]utxi.TxInput{input},
		[]utxi.TxOutput{output},
	}
}