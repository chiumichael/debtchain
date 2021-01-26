package utxi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"strconv"
)

/*
	Note we do not implement versioning, counters, or locktime.
*/
type Transaction struct {
	Inputs		[]TxInput
	Outputs		[]TxOutput
}

func (tx Transaction) String() string {
	var output strings.Builder

	output.WriteString("\nTransaction----\n")
	output.WriteString("Num Inputs: ")

	numInputs := len(tx.Inputs)

	output.WriteString(strconv.Itoa(numInputs))
	output.WriteString("\n")

	for _, v := range tx.Inputs {
		output.WriteString(v.String())
	}

	output.WriteString("Num Outputs: ")
	output.WriteString(strconv.Itoa(len(tx.Outputs)))
	output.WriteString("\n")

	for _, v := range tx.Outputs {
		output.WriteString(v.String())
	}

	return output.String()
}

func (tx *Transaction) IsTransactionValid() bool {
	if (len(tx.Inputs) != len(tx.Outputs)) {
		return false
	}
	return true
}

func (tx *Transaction) IsDebtTransaction() bool {
	if len(tx.Inputs) > 1 {
		return false
	}
	if tx.Inputs[0].Vout == -2 {
		return true
	}
	return false
}

func (debtTx *Transaction) CreateOutstandingDebtTransaction() Transaction {
	return Transaction{Outputs: debtTx.Outputs}
}

func (tx *Transaction) DebtIssued() uint64 {
	var totalIssuedDebt uint64
	totalIssuedDebt = 0

	for _, output := range tx.Outputs {
		totalIssuedDebt = totalIssuedDebt + output.Value
	}
	return totalIssuedDebt
}

func (tx *Transaction) NumInputs() int {
	return len(tx.Inputs)
}

func (tx *Transaction) NumOutputs() int {
	return len(tx.Outputs)
}

func MakeOutstandingDebtTx(tx Transaction) Transaction {
	odtx := tx
	odtx.Inputs = []TxInput{}
	return odtx
}

func (tx *Transaction) Serialize() []byte {
	txAsJson, _ := json.Marshal(tx)
	return txAsJson
}

func (tx *Transaction) Hash() []byte {
	h := sha256.New()
	h.Write(tx.Serialize())
	return h.Sum(nil)
}

func (tx *Transaction) HashStr() string {
	h := sha256.New()
	h.Write(tx.Serialize())
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}