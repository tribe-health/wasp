package chain

import (
	"os"

	"github.com/iotaledger/wasp/client/chainclient"
	"github.com/iotaledger/wasp/packages/coretypes"
	"github.com/iotaledger/wasp/packages/kv/codec"
	"github.com/iotaledger/wasp/packages/sctransaction"
	"github.com/iotaledger/wasp/packages/vm/core/blob"
	"github.com/iotaledger/wasp/packages/vm/core/root"
	"github.com/iotaledger/wasp/tools/wasp-cli/log"
	"github.com/iotaledger/wasp/tools/wasp-cli/util"
)

func deployContractCmd(args []string) {
	if len(args) != 4 {
		log.Fatal("Usage: %s chain deploy-contract <vmtype> <name> <description> <filename>", os.Args[0])
	}

	vmtype := args[0]
	name := args[1]
	description := args[2]
	filename := args[3]

	blobFieldValues := map[string]interface{}{
		blob.VarFieldVMType:             vmtype,
		blob.VarFieldProgramDescription: description,
		blob.VarFieldProgramBinary:      util.ReadFile(filename),
	}

	util.WithSCTransaction(func() (*sctransaction.Transaction, error) {
		return Client().PostRequest(
			blob.Interface.Hname(),
			coretypes.Hn(blob.FuncStoreBlob),
			chainclient.PostRequestParams{
				Args: codec.MakeDict(blobFieldValues),
			},
		)
	})

	progHash := blob.MustGetBlobHash(codec.MakeDict(blobFieldValues))

	util.WithSCTransaction(func() (*sctransaction.Transaction, error) {
		return Client().PostRequest(
			root.Interface.Hname(),
			coretypes.Hn(root.FuncDeployContract),
			chainclient.PostRequestParams{
				Args: codec.MakeDict(map[string]interface{}{
					root.ParamName:        name,
					root.ParamDescription: description,
					root.ParamProgramHash: progHash,
				}),
			},
		)
	})
}
