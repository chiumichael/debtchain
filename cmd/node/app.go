package main

import (
	"encoding/binary"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"debtchain/pkg/utxi"
	"debtchain/internal/envelope"

	"github.com/dgraph-io/badger/v2"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tmlibs/merkle"
)

const (
	codeTypeOK            uint32 = 0
	codeTypeEncodingError uint32 = 1
	codeTypeTicketError   uint32 = 2
)

/*
	Home Equity Loan Backend (HELB)
*/
type HELB struct {
	transactions	*badger.DB
	utxoPool		*badger.DB
	debtPool		*badger.DB
	currentBatch	*badger.Txn
	height			int64
	lastHash		[]byte
	merkletree 		[]merkle.Hasher
}

type ByteWrapper []byte

func (st ByteWrapper) Hash() []byte {
	hasher := sha256.New()
	hasher.Write(st)
	return hasher.Sum(nil)
}

var _ abcitypes.Application = (*HELB)(nil)

func NewHELB(db, utxodb, debtdb *badger.DB) *HELB {
	return &HELB{
		transactions: db,
		utxoPool: utxodb,
		debtPool: debtdb,
		height: 0,
	}
}

func (app *HELB) AddToDebtPool(tx utxi.Transaction) error {
	return app.debtPool.Update(func(txn *badger.Txn) error {
		return txn.Set(tx.Hash(), tx.Serialize())
	})
}

func (app *HELB) AddTransaction(tx utxi.Transaction) error {
	return app.transactions.Update(func(txn *badger.Txn) error {
		return txn.Set(tx.Hash(), tx.Serialize())
	})
}

// we assume proper privacy guidelines are followed that addresses are not reused
// function takes an output and adds 
func (app *HELB) AddToUXTOPool(tx utxi.Transaction) error {
	return app.utxoPool.Update(func(txn *badger.Txn) error {
		for _, output := range tx.Outputs {
			txn.Set(output.RecipientAddr(), output.Serialize())
		}
		return nil
	})
}

func (app *HELB) GetTotalCredits() (error, int) {
	var totalCredits int
	totalCredits = 0
	var output utxi.TxOutput
	err := app.utxoPool.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				jsonErr := json.Unmarshal(v, &output)
				if jsonErr != nil {
					return jsonErr
				}
				totalCredits = totalCredits + int(output.Value)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err, -10
	}
	return nil, totalCredits
}

func (app *HELB) GetTotalDebt() (error, int) {
	var debtAmt int
	debtAmt = 0
	var debtTx utxi.Transaction
	err := app.debtPool.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				jsonErr := json.Unmarshal(v, &debtTx)
				if jsonErr != nil {
					return jsonErr
				}
				debtAmt = debtAmt + int(debtTx.DebtIssued())
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err, -10
	}
	return nil, debtAmt
}

func (app *HELB) HandleRepayment(rpTx utxi.Transaction) error {
	var debtTx utxi.Transaction
	err := app.debtPool.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(rpTx.Inputs[0].Txid)
		if err != nil {
			return err
		}
		valueErr := item.Value(func(v []byte) error {
			jsonErr := json.Unmarshal(v, &debtTx)
			if jsonErr != nil {
				return jsonErr
			}
			return nil
		})
		if valueErr != nil {
			return valueErr
		}
		txn.Delete(rpTx.Inputs[0].Txid)

		debtTx.Outputs[0].Value = debtTx.Outputs[0].Value - rpTx.Outputs[0].Value 
		txn.Set(debtTx.Hash(), debtTx.Serialize())
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (app *HELB) BeginBlock(req abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	app.merkletree = make([]merkle.Hasher, app.height)
	return abcitypes.ResponseBeginBlock{}
}

func (app *HELB) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {

	var cmds envelope.Command
	err := json.Unmarshal(req.Tx, &cmds)
	if err != nil {
		return abcitypes.ResponseCheckTx{
			Code: codeTypeEncodingError,
			Log:  fmt.Sprint(err),
			Info: "Could not parse command JSON",
		}
	}

	switch cmds.Command {
	case "IssueDebt":
		var debtTx utxi.Transaction
		debtTxBytes, _ := base64.RawURLEncoding.DecodeString(cmds.Transaction)
		err := json.Unmarshal(debtTxBytes, &debtTx)

		if err != nil {
			errMsg := fmt.Sprint("error: %v\n", err)
			return abcitypes.ResponseCheckTx{
				Code: 1, 
				GasWanted: 1, 
				Info: errMsg, 
				Data: []byte(cmds.Transaction),
			}
		}
		return abcitypes.ResponseCheckTx{
			Code: 0, 
			GasWanted: 1, 
			Info: strconv.Itoa(int(debtTx.DebtIssued())), 
			Data: []byte("Valid IssueDebt Cmd"),
		}
	case "Repayment":
		var repaymentTx utxi.Transaction
		repaymentTxBytes, _ := base64.RawURLEncoding.DecodeString(cmds.Transaction)
		err := json.Unmarshal(repaymentTxBytes, &repaymentTx)

		if err != nil {
			errMsg := fmt.Sprint("error: %v\n", err)
			return abcitypes.ResponseCheckTx{
				Code: 1, 
				GasWanted: 1, 
				Info: errMsg, 
				Data: []byte(cmds.Transaction),
			}
		}
		return abcitypes.ResponseCheckTx{
			Code: 0, 
			GasWanted: 1, 
			// Info: strconv.Itoa(int(debtTx.DebtIssued())), 
			Data: []byte("Valid Repayment Cmd"),
		}
	}

	return abcitypes.ResponseCheckTx{Code: 0, GasWanted: 1, Info: "unrecognized command", Data: req.Tx}
}

func (app *HELB) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {

	var cmds envelope.Command
	err := json.Unmarshal(req.Tx, &cmds)
	if err != nil {
		return abcitypes.ResponseDeliverTx{
			Code: codeTypeEncodingError,
			Log:  fmt.Sprint(err),
			Info: "Could not parse command JSON",
		}
	}
	switch cmds.Command {
	case "IssueDebt":
		var debtTx utxi.Transaction
		debtTxBytes, _ := base64.RawURLEncoding.DecodeString(cmds.Transaction)
		err := json.Unmarshal(debtTxBytes, &debtTx)
		if err != nil {
			errMsg := fmt.Sprint("error: %v\n", err)
			return abcitypes.ResponseDeliverTx{
				Code: 1, 
				GasWanted: 1, 
				Info: errMsg, 
			}
		}
		
		if (!debtTx.IsTransactionValid()) {
			return abcitypes.ResponseDeliverTx{
				Code: 1, 
				GasWanted: 1, 
				Info: "Invalid Transaction", 
			}
		}
		// transaction has to be in a block
		err = app.AddTransaction(debtTx)
		if err != nil {
			return abcitypes.ResponseDeliverTx{
				Code: 1, 
				GasWanted: 1, 
				Info: fmt.Sprintf("error: %v\n", err),
			}
		}
		app.merkletree = append(app.merkletree, ByteWrapper(debtTx.Hash()))
		// add utxo to utxo pools
		err = app.AddToUXTOPool(debtTx)
		// check for error
		err, totalCredits := app.GetTotalCredits()
		if err != nil {
			return abcitypes.ResponseDeliverTx{
				Code: 1, 
				GasWanted: 1, 
				Info: fmt.Sprintf("error: %v\n", err), 
			}
		}
		// create outstanding dnbt transaction
		odtx := utxi.MakeOutstandingDebtTx(debtTx)
		err = app.AddToDebtPool(odtx)
		if err != nil {
			return abcitypes.ResponseDeliverTx{
				Code: 1, 
				GasWanted: 1, 
				Info: fmt.Sprintf("error: %v\n", err), 
			}
		}
		return abcitypes.ResponseDeliverTx{
			Code: 0,
			GasWanted: 1,
			Info: fmt.Sprintf("Total System Credits: %v", totalCredits),
		}
	case "Repayment":
		var repaymentTx utxi.Transaction
		repaymentTxBytes, _ := base64.RawURLEncoding.DecodeString(cmds.Transaction)
		err := json.Unmarshal(repaymentTxBytes, &repaymentTx)
		if err != nil {
			errMsg := fmt.Sprint("error: %v\n", err)
			return abcitypes.ResponseDeliverTx{
				Code: 1, 
				GasWanted: 1, 
				Info: errMsg, 
			}
		}
		// add to blockchain
		err = app.AddTransaction(repaymentTx)
		if err != nil {
			return abcitypes.ResponseDeliverTx{
				Code: 1, 
				GasWanted: 1, 
				Info: fmt.Sprintf("error: %v\n", err),
			}
		}
		app.merkletree = append(app.merkletree, ByteWrapper(repaymentTx.Hash()))
		// edit the outstanding debt
		err = app.HandleRepayment(repaymentTx)
		if err != nil {
			return abcitypes.ResponseDeliverTx{
				Code: 1, 
				GasWanted: 1, 
				Info: fmt.Sprintf("HandleRepayment Error: %v\n", err),
			}
		}
		// no need to add the utxo's to the utxo pool for repayments
		// querying total system debt
		debtQueryErr, systemDebt:= app.GetTotalDebt()
		if debtQueryErr != nil {
			return abcitypes.ResponseDeliverTx{
				Code: 0,
				GasWanted: 1,
				Info: fmt.Sprintf("error: %v", debtQueryErr),
			}
		}
		return abcitypes.ResponseDeliverTx{
			Code: 0,
			GasWanted: 1,
			Info: fmt.Sprintf("Total System Debt: %v", systemDebt),
		}
	}

	return abcitypes.ResponseDeliverTx{Code: 0}
}

func (app *HELB) Query(reqQuery abcitypes.RequestQuery) (resQuery abcitypes.ResponseQuery) {

	switch reqQuery.Path {
	case "":
		// return abcitypes.ResponseQuery{Value: []byte(fmt.Sprint("here"))}
		return abcitypes.ResponseQuery{Value: []byte(fmt.Sprintf("in nnull case"))}
	case "test":
		return abcitypes.ResponseQuery{Value: []byte(fmt.Sprint("test"))}
		// return abcitypes.ResponseQuery{Value: reqQuery.Data}
	default:
		return abcitypes.ResponseQuery{Value: []byte(fmt.Sprint("couldnt recognize path"))}
	}

	return abcitypes.ResponseQuery{Value: []byte(fmt.Sprint("didn't switch on path"))}
}

func (app *HELB) Commit() abcitypes.ResponseCommit {

	// commented code can be used when Tendermint's empty block generation is diabled
	// if (len(app.merkletree) > 0) {
	// 	blockHash := merkle.SimpleHashFromHashers(app.merkletree)
	// 	app.lastHash = blockHash
	// } else {
	// 	app.lastHash = nila
	// }
	// app.lastHash = merkle.SimpleHashFromHashers(app.merkletree)
	// app.height++
	// return abcitypes.ResponseCommit{Data: app.lastHash}	// merkle root of the block propogated to block tendermint block header

	apphash := make([]byte, 8)
 	binary.PutVarint(apphash, app.height)
 	app.height++
 	return abcitypes.ResponseCommit{Data: apphash}
}

func (HELB) Info(req abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{}
}

func (HELB) SetOption(req abcitypes.RequestSetOption) abcitypes.ResponseSetOption {
	return abcitypes.ResponseSetOption{}
}

func (HELB) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	return abcitypes.ResponseInitChain{}
}

func (app *HELB) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	app.height = req.GetHeight()
	return abcitypes.ResponseEndBlock{}
}
