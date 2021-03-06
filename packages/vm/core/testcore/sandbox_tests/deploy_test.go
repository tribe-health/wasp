package sandbox_tests

import (
	"github.com/iotaledger/wasp/packages/coretypes"
	"github.com/iotaledger/wasp/packages/solo"
	"github.com/iotaledger/wasp/packages/vm/core/root"
	"github.com/iotaledger/wasp/packages/vm/core/testcore/sandbox_tests/test_sandbox_sc"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMainCallsFromFullEP(t *testing.T) {
	_, chain := setupChain(t, nil)
	user := setupDeployer(t, chain)
	userAddress := user.Address()
	userAgentID := coretypes.NewAgentIDFromAddress(userAddress)

	req := solo.NewCall(root.Interface.Name, root.FuncGrantDeploy,
		root.ParamDeployer, userAgentID,
	)
	_, err := chain.PostRequest(req, nil)
	require.NoError(t, err)

	err = chain.DeployContract(user, test_sandbox_sc.Interface.Name, test_sandbox_sc.Interface.ProgramHash)
	require.NoError(t, err)

	contractID := coretypes.NewContractID(chain.ChainID, coretypes.Hn(test_sandbox_sc.Interface.Name))
	agentID := coretypes.NewAgentIDFromContractID(contractID)

	req = solo.NewCall(test_sandbox_sc.Interface.Name, test_sandbox_sc.FuncCheckContextFromFullEP,
		test_sandbox_sc.ParamChainID, chain.ChainID,
		test_sandbox_sc.ParamAgentID, agentID,
		test_sandbox_sc.ParamCaller, userAgentID,
		test_sandbox_sc.ParamChainOwnerID, chain.OriginatorAgentID,
		test_sandbox_sc.ParamContractID, contractID,
		test_sandbox_sc.ParamContractCreator, userAgentID,
	)
	_, err = chain.PostRequest(req, user)
	require.NoError(t, err)
}

func TestMainCallsFromViewEP(t *testing.T) {
	_, chain := setupChain(t, nil)
	user := setupDeployer(t, chain)

	userAddress := user.Address()
	userAgentID := coretypes.NewAgentIDFromAddress(userAddress)

	req := solo.NewCall(root.Interface.Name, root.FuncGrantDeploy,
		root.ParamDeployer, userAgentID,
	)
	_, err := chain.PostRequest(req, nil)
	require.NoError(t, err)

	err = chain.DeployContract(user, test_sandbox_sc.Interface.Name, test_sandbox_sc.Interface.ProgramHash)
	require.NoError(t, err)

	contractID := coretypes.NewContractID(chain.ChainID, coretypes.Hn(test_sandbox_sc.Interface.Name))

	_, err = chain.CallView(test_sandbox_sc.Interface.Name, test_sandbox_sc.FuncCheckContextFromViewEP,
		test_sandbox_sc.ParamChainID, chain.ChainID,
		test_sandbox_sc.ParamContractID, contractID,
	)
	require.NoError(t, err)
}
