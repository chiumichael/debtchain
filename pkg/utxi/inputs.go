package utxi

import (
	"encoding/base64"
	"math/big"
	"strconv"
	"strings"
)

type EcdsaSignature struct {
	R, S 		*big.Int
}

type UnLockingScript struct {
	PublicKey		[]byte
	Signature		EcdsaSignature
}

// TxInputs can be one of three types: CoinbaseInput, DebtInput, or a normal TxInput
type TxInput struct {
	// transaction hash; pointer to the transaction containing the utxo
	Txid				[]byte		
	// index number of the utxo to be spent
	Vout 				int64 		
	// unlocking script (that fullfills condition of the UTXO locking script)
	ScriptSig			UnLockingScript
}

func (txi TxInput) String() string {
	var output strings.Builder
	output.WriteString("Txid:    ")
	output.WriteString(base64.URLEncoding.EncodeToString(txi.Txid))
	output.WriteString("\n")
	output.WriteString("Vout:    ")
	output.WriteString(strconv.Itoa(int(txi.Vout)))
	output.WriteString("\n")
	output.WriteString("UnLockingScript:    ")
	output.WriteString(base64.URLEncoding.EncodeToString(txi.ScriptSig.PublicKey))
	output.WriteString("\n")
	return output.String()
}