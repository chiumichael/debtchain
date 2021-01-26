#!/bin/bash

rm -rf /tmp/debtchain/

TMHOME="/tmp/debtchain/" tendermint init

# remove database artifacts
rm -rf /tmp/badger

mkdir -p /tmp/badger/internal
mkdir -p /tmp/badger/utxo
mkdir -p /tmp/badger/debt
