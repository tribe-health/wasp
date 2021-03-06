package test_sandbox_sc

import (
	"github.com/iotaledger/wasp/packages/coretypes"
	"github.com/iotaledger/wasp/packages/kv"
	"github.com/iotaledger/wasp/packages/kv/codec"
	"github.com/iotaledger/wasp/packages/kv/dict"
	"github.com/iotaledger/wasp/packages/vm/vmtypes"
)

// ParamCallOption
// ParamCallIntParam
// ParamHname
func callOnChain(ctx vmtypes.Sandbox) (dict.Dict, error) {
	ctx.Log().Infof(FuncCallOnChain)
	callOption, exists, err := codec.DecodeString(ctx.Params().MustGet(ParamCallOption))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	if !exists {
		callOption = ""
	}
	callInt, exists, err := codec.DecodeInt64(ctx.Params().MustGet(ParamIntParamValue))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	if !exists {
		ctx.Log().Panicf("parameter '%s' wasn't provided", ParamIntParamValue)
	}
	hname, exists, err := codec.DecodeHname(ctx.Params().MustGet(ParamHname))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	if !exists {
		ctx.Log().Panicf("parameter '%s' wasn't provided", ParamHname)
	}
	ctx.Log().Infof("call depth = %d, option = %s, hname = %s", callInt, callOption, hname)
	if callInt <= 0 {
		return nil, nil
	}
	callInt--

	return ctx.Call(hname, coretypes.Hn(FuncCallOnChain), codec.MakeDict(map[string]interface{}{
		ParamCallOption:    []byte(callOption),
		ParamIntParamValue: callInt,
		ParamHname:         hname,
	}), nil)
}

func getFibonacci(ctx vmtypes.SandboxView) (dict.Dict, error) {
	callInt, exists, err := codec.DecodeInt64(ctx.Params().MustGet(ParamIntParamValue))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	if !exists {
		ctx.Log().Panicf("parameter '%s' wasn't provided", ParamIntParamValue)
	}
	ctx.Log().Infof("fibonacci( %d )", callInt)
	ret := dict.New()
	if callInt == 0 || callInt == 1 {
		ret.Set(ParamIntParamValue, codec.EncodeInt64(callInt))
		return ret, nil
	}
	r1, err := ctx.Call(ctx.ContractID().Hname(), coretypes.Hn(FuncGetFibonacci), codec.MakeDict(map[string]interface{}{
		ParamIntParamValue: callInt - 1,
	}))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	r1val, exists, err := codec.DecodeInt64(r1.MustGet(ParamIntParamValue))
	if err != nil || !exists {
		ctx.Log().Panicf("err != nil || exists #1: %v. %v", exists, err)
	}
	r2, err := ctx.Call(ctx.ContractID().Hname(), coretypes.Hn(FuncGetFibonacci), codec.MakeDict(map[string]interface{}{
		ParamIntParamValue: callInt - 2,
	}))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	r2val, exists, err := codec.DecodeInt64(r2.MustGet(ParamIntParamValue))
	if err != nil || !exists {
		ctx.Log().Panicf("err != nil || !exists #2: %v, %v ", exists, err)
	}
	ret.Set(ParamIntParamValue, codec.EncodeInt64(r1val+r2val))
	return ret, nil
}

// ParamIntParamName
// ParamIntParamValue
func setInt(ctx vmtypes.Sandbox) (dict.Dict, error) {
	ctx.Log().Infof(FuncSetInt)
	paramName, exists, err := codec.DecodeString(ctx.Params().MustGet(ParamIntParamName))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	if !exists {
		ctx.Log().Panicf("parameter '%s' wasn't provided", ParamIntParamName)
	}
	paramValue, exists, err := codec.DecodeInt64(ctx.Params().MustGet(ParamIntParamValue))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	if !exists {
		ctx.Log().Panicf("parameter '%s' wasn't provided", ParamIntParamValue)
	}
	ctx.State().Set(kv.Key(paramName), codec.EncodeInt64(paramValue))
	return nil, nil
}

// ParamIntParamName
func getInt(ctx vmtypes.SandboxView) (dict.Dict, error) {
	ctx.Log().Infof(FuncGetInt)
	paramName, exists, err := codec.DecodeString(ctx.Params().MustGet(ParamIntParamName))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	if !exists {
		ctx.Log().Panicf("parameter '%s' wasn't provided", ParamIntParamName)
	}
	paramValue, exists, err := codec.DecodeInt64(ctx.State().MustGet(kv.Key(paramName)))
	if err != nil {
		ctx.Log().Panicf("%v", err)
	}
	if !exists {
		paramValue = 0
	}
	ret := dict.New()
	ret.Set(kv.Key(paramName), codec.EncodeInt64(paramValue))
	return ret, nil
}
