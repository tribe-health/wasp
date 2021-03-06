// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package vmtypes

import (
	"github.com/iotaledger/wasp/packages/coretypes"
	"github.com/iotaledger/wasp/packages/kv"
	"github.com/iotaledger/wasp/packages/kv/dict"
)

// SandboxView is an interface for read only call
type SandboxView interface {
	// ChainOwnerID AgentID of the current owner of the chain
	ChainOwnerID() coretypes.AgentID
	// ContractCreator agentID which deployed contract
	ContractCreator() coretypes.AgentID
	// ContractID is the ID of the current contract
	ContractID() coretypes.ContractID
	// GetTimestamp return timestamp of the current state
	GetTimestamp() int64
	// Params of the current call
	Params() dict.Dict
	// State immutable k/v store of the current call (in the context of the smart contract)
	State() kv.KVStoreReader
	//Deprecated: -- should be removed FIXME
	WriteableState() kv.KVStore

	// Call calls another contract. Only calls view entry points
	Call(contractHname coretypes.Hname, entryPoint coretypes.Hname, params dict.Dict) (dict.Dict, error)
	// Balances is colored balances owned by the contract
	Balances() coretypes.ColoredBalances
	// Log interface provides local logging on the machine
	Log() LogInterface
}
