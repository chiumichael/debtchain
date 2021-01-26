package utxi 

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
)

type LockingScript struct {
	PubKHash		[]byte
}

type TxOutput struct {
	Value				uint64
	SciptPubKey			LockingScript
}

func ConstructOutput(address []byte, value uint64) TxOutput {
	// assume that the address passed in is a valid public key
	lockscript := LockingScript{address}

	return TxOutput{
		value,
		lockscript,
	}
}

func (tx *TxOutput) Serialize() []byte {
	txAsJson, _ := json.Marshal(tx)
	return txAsJson
}

func (tx *TxOutput) Hash() []byte {
	h := sha256.New()
	h.Write(tx.Serialize())
	return h.Sum(nil)
}

func (tx *TxOutput) HashStr() string {
	h := sha256.New()
	h.Write(tx.Serialize())
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func (tx *TxOutput) RecipientAddr() []byte {
	return tx.SciptPubKey.PubKHash
}

func (tx *TxOutput) RecipientAddrStr() string {
	return base64.URLEncoding.EncodeToString(tx.SciptPubKey.PubKHash)
}

func (txo TxOutput) String() string {
	var output strings.Builder
	output.WriteString("Value:    ")
	output.WriteString(strconv.Itoa(int(txo.Value)))
	output.WriteString("\n")
	output.WriteString("ScriptPubKey:    ")
	output.WriteString(txo.RecipientAddrStr())
	output.WriteString("\n")
	return output.String()
}