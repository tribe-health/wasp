// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package solo

import (
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address/signaturescheme"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/utxodb"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/iotaledger/hive.go/kvstore/mapdb"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/wasp/packages/coretypes"
	"github.com/iotaledger/wasp/packages/sctransaction"
	"github.com/iotaledger/wasp/packages/sctransaction/origin"
	"github.com/iotaledger/wasp/packages/sctransaction/txbuilder"
	"github.com/iotaledger/wasp/packages/state"
	"github.com/iotaledger/wasp/packages/testutil"
	"github.com/iotaledger/wasp/packages/vm"
	"github.com/iotaledger/wasp/packages/vm/processors"
	_ "github.com/iotaledger/wasp/packages/vm/sandbox"
	"github.com/iotaledger/wasp/packages/vm/vmtypes"
	"github.com/iotaledger/wasp/packages/vm/wasmproc"
	"github.com/iotaledger/wasp/plugins/wasmtimevm"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"sync"
	"testing"
	"time"
)

// DefaultTimeStep is a default step for the logical clock for each PostRequest call.
const DefaultTimeStep = 1 * time.Millisecond

// Solo is a structure which contains global parameters of the test: one per test instance
type Solo struct {
	// instance of the test
	T           *testing.T
	logger      *logger.Logger
	utxoDB      *utxodb.UtxoDB
	glbMutex    *sync.Mutex
	logicalTime time.Time
	timeStep    time.Duration
	chains      map[coretypes.ChainID]*Chain
	doOnce      sync.Once
}

// Chain represents state of individual chain.
// There may be several parallel instances of the chain in the 'solo' test
type Chain struct {
	// Env is a pointer to the global structure of the 'solo' test
	Env *Solo

	// Name is the name of the chain
	Name string

	// ChainSigScheme signature scheme of the chain address, the one used to control funds owned by the chain.
	// In Solo it is Ed25519 signature scheme (in full Wasp environment is is a BLS address)
	ChainSigScheme signaturescheme.SignatureScheme

	// OriginatorSigScheme the signature scheme used to create the chain (origin transaction).
	// It is a default signature scheme in many of 'solo' calls which require private key.
	OriginatorSigScheme signaturescheme.SignatureScheme

	// ChainID is the ID of the chain (in this version alias of the ChainAddress)
	ChainID coretypes.ChainID

	// ChainAddress is the alias of ChainSigScheme.Address()
	ChainAddress address.Address

	// ChainColor is the color of the non-fungible token of the chain.
	// It is equal to the hash of the origin transaction of the chain
	ChainColor balance.Color

	// OriginatorAddress is the alias for OriginatorSigScheme.Address()
	OriginatorAddress address.Address

	// OriginatorAgentID is the OriginatorAddress represented in the form of AgentID
	OriginatorAgentID coretypes.AgentID

	// ValidatorFeeTarget is the agent ID to which all fees are accrued. By default is its equal to OriginatorAddress
	ValidatorFeeTarget coretypes.AgentID

	// StateTx is the anchor transaction of the current state of the chain
	StateTx *sctransaction.Transaction

	// State ia an interface to access virtual state of the chain: the collection of key/value pairs
	State state.VirtualState

	// Log is the named logger of the chain
	Log *logger.Logger

	// processor cache
	proc *processors.ProcessorCache

	// related to asynchronous backlog processing
	runVMMutex   *sync.Mutex
	chPosted     sync.WaitGroup
	chInRequest  chan sctransaction.RequestRef
	backlog      []sctransaction.RequestRef
	backlogMutex *sync.Mutex
	batch        []*sctransaction.RequestRef
	batchMutex   *sync.Mutex
}

var (
	doOnce    = sync.Once{}
	glbLogger *logger.Logger
)

// New creates an instance of the `solo` environment for the test instances.
//   'debug' parameter 'true' means logging level is 'debug', otherwise 'info'
//   'printStackTrace' controls printing stack trace in case of errors
func New(t *testing.T, debug bool, printStackTrace bool) *Solo {
	doOnce.Do(func() {
		glbLogger = testutil.NewLogger(t, "04:05.000")
		if !debug {
			glbLogger = testutil.WithLevel(glbLogger, zapcore.InfoLevel, printStackTrace)
		}
		wasmtimeConstructor := func(binary []byte) (vmtypes.Processor, error) {
			return wasmproc.GetProcessor(binary, glbLogger)
		}
		err := processors.RegisterVMType(wasmtimevm.VMType, wasmtimeConstructor)
		require.NoError(t, err)
	})
	ret := &Solo{
		T:           t,
		logger:      glbLogger,
		utxoDB:      utxodb.New(),
		glbMutex:    &sync.Mutex{},
		logicalTime: time.Now(),
		timeStep:    DefaultTimeStep,
		chains:      make(map[coretypes.ChainID]*Chain),
	}
	return ret
}

// NewChain deploys new chain instance.
//
//   If 'chainOriginator' is nil, new one is generated and 1337 iotas is are loaded from the faucet of the UTXODB.
//   If 'validatorFeeTarget' is skipped, it is assumed equal to OriginatorAgentID
// To deploy the chai instance the following steps are performed:
//    - chain signature scheme (private key), chain address and chain ID are created
//    - empty virtual state is initialized
//    - origin transaction is created by the originator and added to the UTXODB
//    - 'init' request transaction to the 'root' contract is created and added to UTXODB
//    - backlog processing threads (goroutines) are started
//    - VM processor cache is initialized
//    - 'init' request is run by the VM. The 'root' contracts deploys the rest of the core contracts:
//      'blob', 'accountsc', 'chainlog'
// Upon return, the chain is fully functional to process requests
func (env *Solo) NewChain(chainOriginator signaturescheme.SignatureScheme, name string, validatorFeeTarget ...coretypes.AgentID) *Chain {
	env.logger.Infof("deploying new chain '%s'", name)
	chSig := signaturescheme.ED25519(ed25519.GenerateKeyPair()) // chain address will be ED25519, not BLS
	if chainOriginator == nil {
		chainOriginator = signaturescheme.ED25519(ed25519.GenerateKeyPair())
		_, err := env.utxoDB.RequestFunds(chainOriginator.Address())
		require.NoError(env.T, err)
	}
	chainID := coretypes.ChainID(chSig.Address())
	originatorAgentID := coretypes.NewAgentIDFromAddress(chainOriginator.Address())
	feeTarget := originatorAgentID
	if len(validatorFeeTarget) > 0 {
		feeTarget = validatorFeeTarget[0]
	}
	ret := &Chain{
		Env:                 env,
		Name:                name,
		ChainSigScheme:      chSig,
		OriginatorSigScheme: chainOriginator,
		ChainAddress:        chSig.Address(),
		OriginatorAddress:   chainOriginator.Address(),
		OriginatorAgentID:   originatorAgentID,
		ValidatorFeeTarget:  feeTarget,
		ChainID:             chainID,
		State:               state.NewVirtualState(mapdb.NewMapDB(), &chainID),
		proc:                processors.MustNew(),
		Log:                 env.logger.Named(name),
		//
		runVMMutex:   &sync.Mutex{},
		chInRequest:  make(chan sctransaction.RequestRef),
		backlog:      make([]sctransaction.RequestRef, 0),
		backlogMutex: &sync.Mutex{},
		batch:        nil,
		batchMutex:   &sync.Mutex{},
	}
	env.AssertAddressBalance(ret.OriginatorAddress, balance.ColorIOTA, testutil.RequestFundsAmount)
	var err error
	ret.StateTx, err = origin.NewOriginTransaction(origin.NewOriginTransactionParams{
		OriginAddress:             ret.ChainAddress,
		OriginatorSignatureScheme: ret.OriginatorSigScheme,
		AllInputs:                 env.utxoDB.GetAddressOutputs(ret.OriginatorAddress),
	})
	require.NoError(env.T, err)
	require.NotNil(env.T, ret.StateTx)
	err = env.utxoDB.AddTransaction(ret.StateTx.Transaction)
	require.NoError(env.T, err)

	ret.ChainColor = balance.Color(ret.StateTx.ID())

	originBlock := state.MustNewOriginBlock(&ret.ChainColor)
	err = ret.State.ApplyBlock(originBlock)
	require.NoError(env.T, err)
	err = ret.State.CommitToDb(originBlock)
	require.NoError(env.T, err)

	initTx, err := origin.NewRootInitRequestTransaction(origin.NewRootInitRequestTransactionParams{
		ChainID:              chainID,
		ChainColor:           ret.ChainColor,
		ChainAddress:         ret.ChainAddress,
		Description:          "'solo' testing chain",
		OwnerSignatureScheme: ret.OriginatorSigScheme,
		AllInputs:            env.utxoDB.GetAddressOutputs(ret.OriginatorAddress),
	})
	require.NoError(env.T, err)
	require.NotNil(env.T, initTx)

	err = env.utxoDB.AddTransaction(initTx.Transaction)
	require.NoError(env.T, err)

	env.glbMutex.Lock()
	env.chains[chainID] = ret
	env.glbMutex.Unlock()

	go ret.readRequestsLoop()
	go ret.batchLoop()

	r := vm.RequestRefWithFreeTokens{}
	r.Tx = initTx
	_, err = ret.runBatch([]vm.RequestRefWithFreeTokens{r}, "new")
	require.NoError(env.T, err)

	ret.Log.Infof("chain '%s' deployed. Chain ID: %s", ret.Name, ret.ChainID)
	return ret
}

func (ch *Chain) readRequestsLoop() {
	for r := range ch.chInRequest {
		ch.backlogMutex.Lock()
		ch.backlog = append(ch.backlog, r)
		ch.backlogMutex.Unlock()
		ch.chPosted.Done()
	}
}

// collateBatch selects requests which are not time locked
// returns batch and and 'remains unprocessed' flag
func (ch *Chain) collateBatch() []vm.RequestRefWithFreeTokens {
	ch.backlogMutex.Lock()
	defer ch.backlogMutex.Unlock()

	ret := make([]vm.RequestRefWithFreeTokens, 0)
	remain := ch.backlog[:0]
	for _, ref := range ch.backlog {
		// using logical clock
		if int64(ref.RequestSection().Timelock()) <= ch.Env.LogicalTime().Unix() {
			if ref.RequestSection().Timelock() != 0 {
				ch.Log.Infof("unlocked time-locked request %s", ref.RequestID().String())
			}
			ret = append(ret, vm.RequestRefWithFreeTokens{RequestRef: ref})
		} else {
			remain = append(remain, ref)
		}
	}
	ch.backlog = remain
	return ret
}

// batchLoop mimics leaders's behavior in the Wasp committee
func (ch *Chain) batchLoop() {
	for {
		batch := ch.collateBatch()
		if len(batch) > 0 {
			_, err := ch.runBatch(batch, "batchLoop")
			if err != nil {
				ch.Log.Errorf("runBatch: %v", err)
			}
			continue
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// backlogLen is a thread-safe function to return size of the current backlog
func (ch *Chain) backlogLen() int {
	ch.chPosted.Wait()
	ch.backlogMutex.Lock()
	defer ch.backlogMutex.Unlock()
	return len(ch.backlog)
}

// NewSignatureSchemeWithFunds generates new ed25519 signature scheme and requests funds (1337 iotas)
// from the UTXODB faucet
func (env *Solo) NewSignatureSchemeWithFunds() signaturescheme.SignatureScheme {
	ret := signaturescheme.ED25519(ed25519.GenerateKeyPair())
	_, err := env.utxoDB.RequestFunds(ret.Address())
	if err != nil {
		env.logger.Panicf("NewSignatureSchemeWithFunds: %v", err)
	}
	env.AssertAddressBalance(ret.Address(), balance.ColorIOTA, testutil.RequestFundsAmount)
	return ret
}

// MintTokens mints specified amount of new colored tokens in the given wallet (signature scheme)
// Returns the color of minted tokens: the hash of the transaction
func (env *Solo) MintTokens(wallet signaturescheme.SignatureScheme, amount int64) (balance.Color, error) {
	allOuts := env.utxoDB.GetAddressOutputs(wallet.Address())
	txb, err := txbuilder.NewFromOutputBalances(allOuts)
	require.NoError(env.T, err)

	if err = txb.MintColor(wallet.Address(), balance.ColorIOTA, amount); err != nil {
		return balance.Color{}, err
	}
	tx := txb.BuildValueTransactionOnly(false)
	tx.Sign(wallet)

	if err = env.utxoDB.AddTransaction(tx); err != nil {
		return balance.Color{}, err
	}
	return balance.Color(tx.ID()), nil
}

// DestroyColoredTokens uncolors specified amount of colored tokens, i.e. converts them into IOTAs
func (env *Solo) DestroyColoredTokens(wallet signaturescheme.SignatureScheme, color balance.Color, amount int64) error {
	allOuts := env.utxoDB.GetAddressOutputs(wallet.Address())
	txb, err := txbuilder.NewFromOutputBalances(allOuts)
	require.NoError(env.T, err)

	if err = txb.EraseColor(wallet.Address(), color, amount); err != nil {
		return err
	}
	tx := txb.BuildValueTransactionOnly(false)
	tx.Sign(wallet)

	return env.utxoDB.AddTransaction(tx)
}
