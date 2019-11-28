package runtime

import block "github.com/filecoin-project/specs/systems/filecoin_blockchain/struct/block"
import filcrypto "github.com/filecoin-project/specs/algorithms/crypto"
import ipld "github.com/filecoin-project/specs/libraries/ipld"
import st "github.com/filecoin-project/specs/systems/filecoin_vm/state_tree"
import msg "github.com/filecoin-project/specs/systems/filecoin_vm/message"
import addr "github.com/filecoin-project/specs/systems/filecoin_vm/actor/address"
import actor "github.com/filecoin-project/specs/systems/filecoin_vm/actor"
import exitcode "github.com/filecoin-project/specs/systems/filecoin_vm/runtime/exitcode"
import util "github.com/filecoin-project/specs/util"

type ActorSubstateCID = actor.ActorSubstateCID
type InvocInput = msg.InvocInput
type InvocOutput = msg.InvocOutput
type ExitCode = exitcode.ExitCode
type RuntimeError = exitcode.RuntimeError

var EnsureErrorCode = exitcode.EnsureErrorCode
var SystemError = exitcode.SystemError
var IMPL_FINISH = util.IMPL_FINISH
var TODO = util.TODO

func ActorSubstateCID_Equals(x, y ActorSubstateCID) bool {
	IMPL_FINISH()
	panic("")
}

// ActorCode is the interface that all actor code types should satisfy.
// It is merely a method dispatch interface.
type ActorCode interface {
	InvokeMethod(rt Runtime, method actor.MethodNum, params actor.MethodParams) InvocOutput
}

type ActorStateHandle struct {
	_initValue *ActorSubstateCID
	_rt        *VMContext
}

func (h *ActorStateHandle) UpdateRelease(newStateCID ActorSubstateCID) {
	h._rt._updateReleaseActorSubstate(newStateCID)
}

func (h *ActorStateHandle) Release(checkStateCID ActorSubstateCID) {
	h._rt._releaseActorSubstate(checkStateCID)
}

func (h *ActorStateHandle) Take() ActorSubstateCID {
	if h._initValue == nil {
		h._rt._apiError("Must call Take() only once on actor substate object")
	}
	ret := *h._initValue
	h._initValue = nil
	return ret
}

// Concrete instantiation of the Runtime interface. This should be instantiated by the
// interpreter once per actor method invocation, and responds to that method's Runtime
// API calls.
type VMContext struct {
	_globalStateInit    st.StateTree
	_globalStatePending st.StateTree
	_running            bool
	_actorAddress       addr.Address
	_actorStateAcquired bool

	_immediateCaller          addr.Address
	_toplevelSender           addr.Address
	_toplevelBlockWinner      addr.Address
	_toplevelSenderCallSeqNum actor.CallSeqNum
	_internalCallSeqNum       actor.CallSeqNum
	_valueReceived            actor.TokenAmount
	_gasRemaining             msg.GasAmount
	_numValidateCalls         int
	_output                   msg.InvocOutput
}

func VMContext_Make(
	toplevelSender addr.Address,
	toplevelBlockWinner addr.Address,
	toplevelSenderCallSeqNum actor.CallSeqNum,
	internalCallSeqNum actor.CallSeqNum,
	globalState st.StateTree,
	actorAddress addr.Address,
	valueReceived actor.TokenAmount,
	gasRemaining msg.GasAmount) *VMContext {

	return &VMContext{
		_globalStateInit:    globalState,
		_globalStatePending: globalState,
		_running:            false,
		_actorAddress:       actorAddress,
		_actorStateAcquired: false,

		_toplevelSender:           toplevelSender,
		_toplevelBlockWinner:      toplevelBlockWinner,
		_toplevelSenderCallSeqNum: toplevelSenderCallSeqNum,
		_internalCallSeqNum:       internalCallSeqNum,
		_valueReceived:            valueReceived,
		_gasRemaining:             gasRemaining,
		_numValidateCalls:         0,
		_output:                   nil,
	}
}

func (rt *VMContext) AbortArgMsg(msg string) Runtime_AbortArgMsg_FunRet {
	rt.Abort(exitcode.UserDefinedError(exitcode.InvalidArguments_User), msg)
	return &Runtime_AbortArgMsg_FunRet_I{}
}

func (rt *VMContext) AbortArg() Runtime_AbortArg_FunRet {
	rt.AbortArgMsg("Invalid arguments")
	return &Runtime_AbortArg_FunRet_I{}
}

func (rt *VMContext) AbortStateMsg(msg string) Runtime_AbortStateMsg_FunRet {
	rt.Abort(exitcode.UserDefinedError(exitcode.InconsistentState_User), msg)
	return &Runtime_AbortStateMsg_FunRet_I{}
}

func (rt *VMContext) AbortState() Runtime_AbortState_FunRet {
	rt.AbortStateMsg("Inconsistent state")
	return &Runtime_AbortState_FunRet_I{}
}

func (rt *VMContext) AbortFundsMsg(msg string) Runtime_AbortFundsMsg_FunRet {
	rt.Abort(exitcode.UserDefinedError(exitcode.InsufficientFunds_User), msg)
	return &Runtime_AbortFundsMsg_FunRet_I{}
}

func (rt *VMContext) AbortFunds() Runtime_AbortFunds_FunRet {
	rt.AbortFundsMsg("Insufficient funds")
	return &Runtime_AbortFunds_FunRet_I{}
}

func (rt *VMContext) AbortAPI(msg string) Runtime_AbortAPI_FunRet {
	rt.Abort(exitcode.SystemError(exitcode.RuntimeAPIError), msg)
	return &Runtime_AbortAPI_FunRet_I{}
}

func (rt *VMContext) CreateActor(
	stateCID actor.ActorSystemStateCID,
	address addr.Address,
	initBalance actor.TokenAmount,
	constructorParams actor.MethodParams) Runtime_CreateActor_FunRet {

	if !rt._actorAddress.Equals(addr.InitActorAddr) {
		rt.AbortAPI("Only InitActor may call rt.CreateActor")
	}

	rt._updateActorSystemStateInternal(address, stateCID)

	rt.SendPropagatingErrors(&msg.InvocInput_I{
		To_:     address,
		Method_: actor.MethodConstructor,
		Params_: constructorParams,
		Value_:  initBalance,
	})

	return &Runtime_CreateActor_FunRet_I{}
}

func (rt *VMContext) _updateActorSystemStateInternal(actorAddress addr.Address, newStateCID actor.ActorSystemStateCID) {
	newGlobalStatePending, err := rt._globalStatePending.Impl().WithActorSystemState(rt._actorAddress, newStateCID)
	if err != nil {
		panic("Error in runtime implementation: failed to update actor system state")
	}
	rt._globalStatePending = newGlobalStatePending
}

func (rt *VMContext) _updateActorSubstateInternal(actorAddress addr.Address, newStateCID actor.ActorSubstateCID) {
	newGlobalStatePending, err := rt._globalStatePending.Impl().WithActorSubstate(rt._actorAddress, newStateCID)
	if err != nil {
		panic("Error in runtime implementation: failed to update actor substate")
	}
	rt._globalStatePending = newGlobalStatePending
}

func (rt *VMContext) _updateReleaseActorSubstate(newStateCID ActorSubstateCID) {
	rt._checkRunning()
	rt._checkActorStateAcquired()
	rt._updateActorSubstateInternal(rt._actorAddress, newStateCID)
	rt._actorStateAcquired = false
}

func (rt *VMContext) _releaseActorSubstate(checkStateCID ActorSubstateCID) {
	rt._checkRunning()
	rt._checkActorStateAcquired()

	prevState := rt._globalStatePending.GetActorState(rt._actorAddress)
	prevStateCID := prevState.State()
	if !ActorSubstateCID_Equals(prevStateCID, checkStateCID) {
		rt.AbortAPI("State CID differs upon release call")
	}

	rt._actorStateAcquired = false
}

func (rt *VMContext) Assert(cond bool) Runtime_Assert_FunRet {
	if !cond {
		rt.Abort(exitcode.SystemError(exitcode.RuntimeAssertFailure), "Runtime assertion failed")
	}
	return &Runtime_Assert_FunRet_I{}
}

func (rt *VMContext) _checkActorStateAcquiredFlag(expected bool) {
	rt._checkRunning()
	if rt._actorStateAcquired != expected {
		rt._apiError("State updates and message sends must be disjoint")
	}
}

func (rt *VMContext) _checkActorStateAcquired() {
	rt._checkActorStateAcquiredFlag(true)
}

func (rt *VMContext) _checkActorStateNotAcquired() {
	rt._checkActorStateAcquiredFlag(false)
}

func (rt *VMContext) Abort(errExitCode exitcode.ExitCode, errMsg string) Runtime_Abort_FunRet {
	errExitCode = exitcode.EnsureErrorCode(errExitCode)
	rt._throwErrorFull(errExitCode, errMsg)
	return &Runtime_Abort_FunRet_I{}
}

func (rt *VMContext) ImmediateCaller() addr.Address {
	return rt._immediateCaller
}

func (rt *VMContext) ToplevelSender() addr.Address {
	return rt._toplevelSender
}

func (rt *VMContext) ToplevelBlockWinner() addr.Address {
	return rt._toplevelBlockWinner
}

func (rt *VMContext) InternalCallSeqNum() actor.CallSeqNum {
	return rt._internalCallSeqNum
}

func (rt *VMContext) ToplevelSenderCallSeqNum() actor.CallSeqNum {
	return rt._toplevelSenderCallSeqNum
}

func (rt *VMContext) ValidateImmediateCallerMatches(
	callerExpectedPattern CallerPattern) Runtime_ValidateImmediateCallerMatches_FunRet {

	rt._checkRunning()
	rt._checkNumValidateCalls(0)
	caller := rt.ImmediateCaller()
	if !callerExpectedPattern.Matches(caller) {
		rt.AbortAPI("Method invoked by incorrect caller")
	}
	rt._numValidateCalls += 1
	return &Runtime_ValidateImmediateCallerMatches_FunRet_I{}
}

type CallerPattern struct {
	Matches func(addr.Address) bool
}

func CallerPattern_MakeSingleton(x addr.Address) CallerPattern {
	return CallerPattern{
		Matches: func(y addr.Address) bool { return x == y },
	}
}

func CallerPattern_MakeAcceptAny() CallerPattern {
	return CallerPattern{
		Matches: func(addr.Address) bool { return true },
	}
}

func (rt *VMContext) ValidateImmediateCallerIs(callerExpected addr.Address) Runtime_ValidateImmediateCallerIs_FunRet {
	rt.ValidateImmediateCallerMatches(CallerPattern_MakeSingleton(callerExpected))
	return &Runtime_ValidateImmediateCallerIs_FunRet_I{}
}

func (rt *VMContext) ValidateImmediateCallerAcceptAny() Runtime_ValidateImmediateCallerAcceptAny_FunRet {
	rt.ValidateImmediateCallerMatches(CallerPattern_MakeAcceptAny())
	return &Runtime_ValidateImmediateCallerAcceptAny_FunRet_I{}
}

func (rt *VMContext) _checkNumValidateCalls(x int) {
	if rt._numValidateCalls != x {
		rt.AbortAPI("Method must validate caller identity exactly once")
	}
}

func (rt *VMContext) _checkRunning() {
	if !rt._running {
		panic("Internal runtime error: actor API called with no actor code running")
	}
}
func (rt *VMContext) SuccessReturn() InvocOutput {
	return msg.InvocOutput_Make(nil)
}

func (rt *VMContext) ValueReturn(value util.Bytes) InvocOutput {
	return msg.InvocOutput_Make(value)
}

func (rt *VMContext) _throwError(exitCode ExitCode) {
	rt._throwErrorFull(exitCode, "")
}

func (rt *VMContext) _throwErrorFull(exitCode ExitCode, errMsg string) {
	panic(exitcode.RuntimeError_Make(exitCode, errMsg))
}

func (rt *VMContext) _apiError(errMsg string) {
	rt._throwErrorFull(exitcode.SystemError(exitcode.RuntimeAPIError), errMsg)
}

func (rt *VMContext) _checkGasRemaining() {
	if rt._gasRemaining.LessThan(msg.GasAmount_Zero()) {
		rt._throwError(exitcode.SystemError(exitcode.OutOfGas))
	}
}

func (rt *VMContext) _deductGasRemaining(x msg.GasAmount) {
	// TODO: check x >= 0
	rt._checkGasRemaining()
	rt._gasRemaining = rt._gasRemaining.Subtract(x)
	rt._checkGasRemaining()
}

func (rt *VMContext) _refundGasRemaining(x msg.GasAmount) {
	// TODO: check x >= 0
	rt._checkGasRemaining()
	rt._gasRemaining = rt._gasRemaining.Add(x)
	rt._checkGasRemaining()
}

func (rt *VMContext) _transferFunds(from addr.Address, to addr.Address, amount actor.TokenAmount) error {
	rt._checkRunning()
	rt._checkActorStateNotAcquired()

	newGlobalStatePending, err := rt._globalStatePending.Impl().WithFundsTransfer(from, to, amount)
	if err != nil {
		return err
	}

	rt._globalStatePending = newGlobalStatePending
	return nil
}

type ErrorHandlingSpec int

const (
	PropagateErrors ErrorHandlingSpec = 1 + iota
	CatchErrors
)

// TODO: This function should be private (not intended to be exposed to actors).
// (merging runtime and interpreter packages should solve this)
func (rt *VMContext) SendToplevelFromInterpreter(input InvocInput) (
	msg.MessageReceipt, st.StateTree) {

	rt._running = true
	ret := rt._sendInternal(input, CatchErrors)
	rt._running = false
	return ret, rt._globalStatePending
}

func _catchRuntimeErrors(f func() msg.InvocOutput) (output msg.InvocOutput, exitCode exitcode.ExitCode) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case *RuntimeError:
				output = msg.InvocOutput_Make(nil)
				exitCode = (r.(*RuntimeError).ExitCode)
			default:
				panic(r)
			}
		}
	}()

	output = f()
	exitCode = exitcode.OK()
	return
}

func _invokeMethodInternal(
	rt *VMContext,
	actorCode ActorCode,
	method actor.MethodNum,
	params actor.MethodParams) (
	ret InvocOutput, gasUsed msg.GasAmount, exitCode exitcode.ExitCode, internalCallSeqNumFinal actor.CallSeqNum) {

	if method == actor.MethodSend {
		gasUsed = msg.GasAmount_Zero() // TODO: verify
		ret = msg.InvocOutput_Make(nil)
		return
	}

	rt._running = true
	ret, exitCode = _catchRuntimeErrors(func() InvocOutput {
		methodOutput := actorCode.InvokeMethod(rt, method, params)
		rt._checkActorStateNotAcquired()
		rt._checkNumValidateCalls(1)
		return methodOutput
	})
	rt._running = false

	// TODO: Update gasUsed
	TODO()

	internalCallSeqNumFinal = rt._internalCallSeqNum

	return
}

func (rtOuter *VMContext) _sendInternal(input InvocInput, errSpec ErrorHandlingSpec) msg.MessageReceipt {
	rtOuter._checkRunning()
	rtOuter._checkActorStateNotAcquired()

	toActor := rtOuter._globalStatePending.GetActorState(input.To())

	toActorCode, err := loadActorCode(toActor.CodeID())
	if err != nil {
		rtOuter._throwError(exitcode.SystemError(exitcode.ActorCodeNotFound))
	}

	var toActorMethodGasBound msg.GasAmount
	TODO() // TODO: obtain from actor registry
	rtOuter._deductGasRemaining(toActorMethodGasBound)
	// TODO: gasUsed may be larger than toActorMethodGasBound if toActor itself makes sub-calls.
	// To prevent this, we would need to calculate the gas bounds recursively.

	err = rtOuter._transferFunds(rtOuter._actorAddress, input.To(), input.Value())
	if err != nil {
		rtOuter._throwError(exitcode.SystemError(exitcode.InsufficientFunds_System))
	}

	rtInner := VMContext_Make(
		rtOuter._toplevelSender,
		rtOuter._toplevelBlockWinner,
		rtOuter._toplevelSenderCallSeqNum,
		rtOuter._internalCallSeqNum+1,
		rtOuter._globalStatePending,
		input.To(),
		input.Value(),
		rtOuter._gasRemaining,
	)

	invocOutput, gasUsed, exitCode, internalCallSeqNumFinal := _invokeMethodInternal(
		rtInner,
		toActorCode,
		input.Method(),
		input.Params(),
	)

	rtOuter._internalCallSeqNum = internalCallSeqNumFinal

	rtOuter._refundGasRemaining(toActorMethodGasBound)
	rtOuter._deductGasRemaining(gasUsed)
	if exitCode.Equals(exitcode.SystemError(exitcode.OutOfGas)) {
		// OutOfGas error cannot be caught
		rtOuter._throwError(exitCode)
	}

	if errSpec == PropagateErrors && exitCode.IsError() {
		rtOuter._throwError(exitcode.SystemError(exitcode.MethodSubcallError))
	}

	if exitCode.AllowsStateUpdate() {
		rtOuter._globalStatePending = rtInner._globalStatePending
	}

	return msg.MessageReceipt_Make(invocOutput, exitCode, gasUsed)
}

func (rtOuter *VMContext) _sendInternalOutputs(input InvocInput, errSpec ErrorHandlingSpec) (msg.InvocOutput, exitcode.ExitCode) {
	ret := rtOuter._sendInternal(input, errSpec)
	return msg.InvocOutput_Make(ret.ReturnValue()), ret.ExitCode()
}

func (rt *VMContext) SendPropagatingErrors(input InvocInput) msg.InvocOutput {
	ret, _ := rt._sendInternalOutputs(input, PropagateErrors)
	return ret
}

func (rt *VMContext) SendCatchingErrors(input InvocInput) (msg.InvocOutput, exitcode.ExitCode) {
	return rt._sendInternalOutputs(input, CatchErrors)
}

func (rt *VMContext) CurrentBalance() actor.TokenAmount {
	IMPL_FINISH()
	panic("")
}

func (rt *VMContext) ValueReceived() actor.TokenAmount {
	return rt._valueReceived
}

func (rt *VMContext) Randomness(e block.ChainEpoch, offset uint64) util.Randomness {
	// TODO: validate CurrEpoch() - K <= e <= CurrEpoch()?
	// TODO: finish
	TODO()
	panic("")
}

func (rt *VMContext) IpldPut(x ipld.Object) ipld.CID {
	panic("TODO")
}

func (rt *VMContext) IpldGet(c ipld.CID) Runtime_IpldGet_FunRet {
	panic("TODO")
}

func (rt *VMContext) CurrEpoch() block.ChainEpoch {
	IMPL_FINISH()
	panic("")
}

func (rt *VMContext) AcquireState() ActorStateHandle {
	rt._checkRunning()
	rt._checkActorStateNotAcquired()
	rt._actorStateAcquired = true

	return ActorStateHandle{
		_initValue: rt._globalStatePending.GetActorState(rt._actorAddress).State().Ref(),
		_rt:        rt,
	}
}

func (rt *VMContext) CurrMethodNum() actor.MethodNum {
	IMPL_FINISH()
	panic("")
}

func (rt *VMContext) VerifySignature(signerActor addr.Address, sig filcrypto.Signature, m filcrypto.Message) bool {
	st := rt._globalStatePending.Impl().GetActorState(signerActor)
	if st == nil {
		rt.AbortAPI("VerifySignature: signer actor not found")
	}
	pk := st.GetSignaturePublicKey()
	if pk == nil {
		rt.AbortAPI("VerifySignature: signer actor has no public key")
	}
	ret := rt.Compute(ComputeFunctionID_VerifySignature, []Any{pk, sig, m})
	return ret.(bool)
}

func (rt *VMContext) Compute(f ComputeFunctionID, args []Any) Any {
	def, found := _computeFunctionDefs[f]
	if !found {
		rt.AbortAPI("Function definition in rt.Compute() not found")
	}
	gasCost := def.GasCostFn(args)
	rt._deductGasRemaining(gasCost)
	return def.Body(args)
}
