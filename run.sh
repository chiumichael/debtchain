#!/bin/bash

# Command: IssueDebt
# Address: "eb5c1"
# Amount: 10
curl --data-binary '{"jsonrpc":"2.0","id":"anything","method":"broadcast_tx_commit","params": {"tx": "eyJDb21tYW5kIjoiSXNzdWVEZWJ0IiwiQWRkcmVzcyI6ImViNWMxIiwiQW1vdW50IjoxMH0="}}' -H 'content-type:text/plain;' http://localhost:26657
