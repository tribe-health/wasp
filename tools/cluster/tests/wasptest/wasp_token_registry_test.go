package wasptest

import (
	"github.com/iotaledger/wasp/packages/hashing"
	"github.com/iotaledger/wasp/packages/vm/examples/tokenregistry"
	"testing"
	"time"

	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/wasp/packages/kv"
	"github.com/iotaledger/wasp/packages/testutil"
	"github.com/iotaledger/wasp/packages/vm/vmconst"
)

const scTokenRegistryNum = 6

// sending 5 NOP requests with 1 sec sleep between each
func TestTRMint1Token(t *testing.T) {
	// setup
	wasps := setup(t, "test_cluster", "TestSC6Requests5Sec1")

	err := wasps.ListenToMessages(map[string]int{
		"bootuprec":           1, // wasps.NumSmartContracts(),
		"active_committee":    1,
		"dismissed_committee": 0,
		"request_in":          2,
		"request_out":         3,
		"state":               -1, // must be 6 or 7
		"vmmsg":               -1,
	})
	check(err, t)

	// number 5 is "Wasm VM PoC program" in cluster.json
	sc := &wasps.SmartContractConfig[scTokenRegistryNum]

	_, err = PutBootupRecord(wasps, sc)
	check(err, t)

	err = Activate1SC(wasps, sc)
	check(err, t)

	err = CreateOrigin1SC(wasps, sc)
	check(err, t)

	time.Sleep(2 * time.Second)

	scOwnerAddr := sc.OwnerAddress()
	scAddress := sc.SCAddress()
	scColor := sc.GetColor()
	minter1Addr := minter1.Address()
	progHash, err := hashing.HashValueFromBase58(tokenregistry.ProgramHash)
	check(err, t)

	err = wasps.NodeClient.RequestFunds(&minter1Addr)
	check(err, t)

	time.Sleep(2 * time.Second)

	if !wasps.VerifyAddressBalances(minter1Addr, testutil.RequestFundsAmount, map[balance.Color]int64{
		balance.ColorIOTA: testutil.RequestFundsAmount,
	}, "minter1 in the beginning") {
		t.Fail()
		return
	}
	if !wasps.VerifyAddressBalances(scAddress, 1, map[balance.Color]int64{
		scColor: 1, // sc token
	}, "SC address in the beginning") {
		t.Fail()
		return
	}
	if !wasps.VerifyAddressBalances(scOwnerAddr, testutil.RequestFundsAmount-1, map[balance.Color]int64{
		balance.ColorIOTA: testutil.RequestFundsAmount - 1,
	}, "owner in the beginning") {
		t.Fail()
		return
	}

	mintedColor1, err := tokenregistry.MintAndRegister(wasps.NodeClient, tokenregistry.MintAndRegisterParams{
		SenderSigScheme: minter1.SigScheme(),
		Supply:          1,
		MintTarget:      minter1Addr,
		RegistryAddr:    scAddress,
		Description:     "Non-fungible coin 1",
	})
	check(err, t)

	//wasps.CollectMessages(30 * time.Second)
	wasps.WaitUntilExpectationsMet()

	if !wasps.Report() {
		t.Fail()
	}

	if !wasps.VerifyAddressBalances(scAddress, 1, map[balance.Color]int64{
		balance.ColorIOTA: 0,
		sc.GetColor():     1,
	}, "SC address in the end") {
		t.Fail()
	}

	if !wasps.VerifyAddressBalances(minter1Addr, testutil.RequestFundsAmount, map[balance.Color]int64{
		*mintedColor1:     1,
		balance.ColorIOTA: testutil.RequestFundsAmount - 1,
	}, "minter1 in the end") {
		t.Fail()
		return
	}

	if !wasps.VerifySCStateVariables(sc, map[kv.Key][]byte{
		vmconst.VarNameOwnerAddress:      scOwnerAddr.Bytes(),
		vmconst.VarNameProgramHash:       progHash.Bytes(),
		tokenregistry.VarStateListColors: []byte(mintedColor1.String()),
	}) {
		t.Fail()
	}
}
