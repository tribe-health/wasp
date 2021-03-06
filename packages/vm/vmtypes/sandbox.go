// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package vmtypes

import (
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/wasp/packages/coretypes"
	"github.com/iotaledger/wasp/packages/hashing"
	"github.com/iotaledger/wasp/packages/kv"
	"github.com/iotaledger/wasp/packages/kv/dict"
)

// Sandbox is an interface given to the processor to access the VMContext
// and virtual state, transaction builder and request parameters through it.
type Sandbox interface {
	// ChainOwnerID AgentID of the current owner of the chain
	ChainOwnerID() coretypes.AgentID
	// ContractCreator agentID which deployed contract
	ContractCreator() coretypes.AgentID
	// ContractID is the ID of the current contract
	ContractID() coretypes.ContractID
	// State is base level interface to access the key/value pairs in the virtual state
	// GetTimestamp return current timestamp of the context
	GetTimestamp() int64
	// Params of the current call
	Params() dict.Dict
	// State k/v store of the current call (in the context of the smart contract)
	State() kv.KVStore

	// Caller is the agentID of the caller of he SC function
	Caller() coretypes.AgentID

	// CreateContract deploys contract on the same chain. 'initParams' are passed to the 'init' entry point
	DeployContract(programHash hashing.HashValue, name string, description string, initParams dict.Dict) error
	// Call calls the entry point of the contract with parameters and transfer.
	// If the entry point is full entry point, transfer tokens are moved between caller's and
	// target contract's accounts (if enough)
	// If the entry point is view, 'transfer' has no effect
	Call(target coretypes.Hname, entryPoint coretypes.Hname, params dict.Dict, transfer coretypes.ColoredBalances) (dict.Dict, error)
	// RequestID of the request in the context of which is the current call
	RequestID() coretypes.RequestID
	// GetEntropy 32 random bytes based on the hash of the current state transaction
	GetEntropy() hashing.HashValue // 32 bytes of deterministic and unpredictably random data

	// Access to balances and tokens
	// Balances returns colored balances owned by the smart contract
	Balances() coretypes.ColoredBalances
	// IncomingTransfer return colored balances transferred by the call. They are already accounted into the Balances()
	IncomingTransfer() coretypes.ColoredBalances
	// Balance return number of tokens of specific color in the balance of the smart contract
	Balance(col balance.Color) int64

	// MoveTokens moves specified colored tokens to the target account on the same chain
	// Deprecated: equivalent to calling "deposit" to "accounts" on the same chain
	MoveTokens(target coretypes.AgentID, col balance.Color, amount int64) bool

	// Moving tokens outside of the current chain
	// TransferToAddress send tokens to the L1 ledger address
	TransferToAddress(addr address.Address, transfer coretypes.ColoredBalances) bool

	// TransferCrossChain send funds to the targetAgentID account cross chain
	// syntactic sugar for sending "deposit" request to the "accounts" contract on the target chain
	// Deprecated: it is just a syntactic sugar for PostRequest "deposit" to accounts
	TransferCrossChain(targetAgentID coretypes.AgentID, targetChainID coretypes.ChainID, transfer coretypes.ColoredBalances) bool
	// PostRequest sends cross-chain request
	PostRequest(par PostRequestParams) bool

	// Log interface provides local logging on the machine
	Log() LogInterface
	// Event publishes "vmmsg" message through Publisher on nanomsg
	// it also logs locally, but it is not the same thing
	Event(msg string)
}

type PostRequestParams struct {
	TargetContractID coretypes.ContractID
	EntryPoint       coretypes.Hname
	TimeLock         uint32
	Params           dict.Dict
	Transfer         coretypes.ColoredBalances
}

type LogInterface interface {
	Infof(format string, param ...interface{})
	Debugf(format string, param ...interface{})
	Panicf(format string, param ...interface{})
}
