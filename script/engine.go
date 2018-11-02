// Copyright (c) 2013-2018 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/log"
	"github.com/elastos/Elastos.ELA/core"
)

const (
	// MaxStackSize is the maximum combined height of stack and alt stack
	// during execution.
	MaxStackSize = 1000

	// MaxScriptSize is the maximum allowed length of a raw script.
	MaxScriptSize = 10000
)

// Engine is the virtual machine that executes scripts.
type Engine struct {
	scripts         [][]parsedOpcode
	scriptIdx       int
	scriptOff       int
	lastCodeSep     int
	dstack          stack // data stack
	astack          stack // alt stack
	condStack       []int
	numOps          int
	inputAmount     int64
	bip16           bool     // treat execution as pay-to-script-hash
	savedFirstStack [][]byte // stack from first script for bip16 scripts
	tx              core.Transaction
	txIdx           int
}

// isBranchExecuting returns whether or not the current conditional branch is
// actively executing.  For example, when the data stack has an OP_FALSE on it
// and an OP_IF is encountered, the branch is inactive until an OP_ELSE or
// OP_ENDIF is encountered.  It properly handles nested conditionals.
func (vm *Engine) isBranchExecuting() bool {
	if len(vm.condStack) == 0 {
		return true
	}
	return vm.condStack[len(vm.condStack)-1] == OpCondTrue
}

// popIfBool enforces the "minimal if" policy during script execution if the
// particular flag is set.  If so, in order to eliminate an additional source
// of nuisance malleability, post-segwit for version 0 witness programs, we now
// require the following: for OP_IF and OP_NOT_IF, the top stack item MUST
// either be an empty byte slice, or [0x01]. Otherwise, the item at the top of
// the stack will be popped and interpreted as a boolean.
func popIfBool(vm *Engine) (bool, error) {
	return vm.dstack.PopBool()
}

// getStack returns the contents of stack as a byte array bottom up
func getStack(stack *stack) [][]byte {
	array := make([][]byte, stack.Depth())
	for i := range array {
		// PeekByteArry can't fail due to overflow, already checked
		array[len(array)-i-1], _ = stack.PeekByteArray(int32(i))
	}
	return array
}

// setStack sets the stack to the contents of the array where the last item in
// the array is the top item in the stack.
func setStack(stack *stack, data [][]byte) {
	// This can not error. Only errors are for invalid arguments.
	_ = stack.DropN(stack.Depth())

	for i := range data {
		stack.PushByteArray(data[i])
	}
}

// GetStack returns the contents of the primary stack as an array. where the
// last item in the array is the top of the stack.
func (vm *Engine) GetStack() [][]byte {
	return getStack(&vm.dstack)
}

// SetStack sets the contents of the primary stack to the contents of the
// provided array where the last item in the array will be the top of the stack.
func (vm *Engine) SetStack(data [][]byte) {
	setStack(&vm.dstack, data)
}

// GetAltStack returns the contents of the alternate stack as an array where the
// last item in the array is the top of the stack.
func (vm *Engine) GetAltStack() [][]byte {
	return getStack(&vm.astack)
}

// SetAltStack sets the contents of the alternate stack to the contents of the
// provided array where the last item in the array will be the top of the stack.
func (vm *Engine) SetAltStack(data [][]byte) {
	setStack(&vm.astack, data)
}

func NewEngine(script []byte, tx *core.Transaction, txIdx int) (*Engine, error) {
	if len(script) == 0 {
		return nil, errors.New("false stack entry at end of script execution")
	}

	// The provided transaction input index must refer to a valid input.
	if txIdx < 0 || txIdx >= len(tx.Inputs) {
		str := fmt.Sprintf("transaction input index %d is negative or "+
			">= %d", txIdx, len(tx.Inputs))
		return nil, scriptError(ErrInvalidIndex, str)
	}

	vm := Engine{}

	scripts := [][]byte{script}
	vm.scripts = make([][]parsedOpcode, len(scripts))
	vm.tx = *tx
	vm.txIdx = txIdx

	for i, scr := range scripts {
		if len(scr) > MaxScriptSize {
			str := fmt.Sprintf("script size %d is larger than max "+
				"allowed size %d", len(scr), MaxScriptSize)
			return nil, errors.New(str)
		}
		var err error
		vm.scripts[i], err = parseScript(scr)
		if err != nil {
			return nil, err
		}
	}
	return &vm, nil
}

// disasm is a helper function to produce the output for DisasmPC and
// DisasmScript.  It produces the opcode prefixed by the program counter at the
// provided position in the script.  It does no error checking and leaves that
// to the caller to provide a valid offset.
func (vm *Engine) disasm(scriptIdx int, scriptOff int) string {
	return fmt.Sprintf("%02x:%04x: %s", scriptIdx, scriptOff,
		vm.scripts[scriptIdx][scriptOff].print(false))
}

// DisasmScript returns the disassembly string for the script at the requested
// offset index.  Index 0 is the signature script and 1 is the public key
// script.
func (vm *Engine) DisasmScript(idx int) (string, error) {
	if idx >= len(vm.scripts) {
		str := fmt.Sprintf("script index %d >= total scripts %d", idx,
			len(vm.scripts))
		return "", errors.New(str)
	}

	var disstr string
	for i := range vm.scripts[idx] {
		disstr = disstr + vm.disasm(idx, i) + "\n"
	}
	return disstr, nil
}

// CheckErrorCondition returns nil if the running script has ended and was
// successful, leaving a a true boolean on the stack.  An error otherwise,
// including if the script has not finished.
func (vm *Engine) CheckErrorCondition(finalScript bool) error {
	// Check execution is actually done.  When pc is past the end of script
	// array there are no more scripts to run.
	if vm.scriptIdx < len(vm.scripts) {
		return scriptError(ErrScriptUnfinished, "error check when script unfinished")
	}

	v, err := vm.dstack.PopBool()
	if err != nil {
		return err
	}
	if !v {
		dis0, _ := vm.DisasmScript(0)
		dis1, _ := vm.DisasmScript(1)
		str := fmt.Sprintf("scripts failed: script0: %s\n"+
			"script1: %s", dis0, dis1)
		log.Info(str)
		return scriptError(ErrEvalFalse, "false stack entry at end of script execution")
	}
	return nil
}

// Step will execute the next instruction and move the program counter to the
// next opcode in the script, or the next script if the current has ended.  Step
// will return true in the case that the last opcode was successfully executed.
//
// The result of calling Step or any other method is undefined if an error is
// returned.
func (vm *Engine) Step() (done bool, err error) {
	// Verify that it is pointing to a valid script address.
	err = vm.validPC()
	if err != nil {
		return true, err
	}
	opcode := &vm.scripts[vm.scriptIdx][vm.scriptOff]
	vm.scriptOff++
	// Execute the opcode while taking into account several things such as
	// disabled opcodes, illegal opcodes, maximum allowed operations per
	// script, maximum script element sizes, and conditionals.
	err = vm.executeOpcode(opcode)
	if err != nil {
		return true, err
	}

	// The number of elements in the combination of the data and alt stacks
	// must not exceed the maximum number of stack elements allowed.
	combinedStackSize := vm.dstack.Depth() + vm.astack.Depth()
	if combinedStackSize > MaxStackSize {
		str := fmt.Sprintf("combined stack size %d > max allowed %d",
			combinedStackSize, MaxStackSize)
		return false, errors.New(str)
	}

	// Prepare for next instruction.
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
		// Illegal to have an `if' that straddles two scripts.
		if err == nil && len(vm.condStack) != 0 {
			return false, errors.New("end of script reached in conditional execution")
		}

		// Alt stack doesn't persist.
		_ = vm.astack.DropN(vm.astack.Depth())

		vm.numOps = 0 // number of ops is per script.
		vm.scriptOff = 0
		if vm.scriptIdx == 0 && vm.bip16 {
			vm.scriptIdx++
			vm.savedFirstStack = vm.GetStack()
		} else if vm.scriptIdx == 1 && vm.bip16 {
			// Put us past the end for CheckErrorCondition()
			vm.scriptIdx++
			// Check script ran successfully and pull the script
			// out of the first stack and execute that.
			err := vm.CheckErrorCondition(false)
			if err != nil {
				return false, err
			}

			script := vm.savedFirstStack[len(vm.savedFirstStack)-1]
			pops, err := parseScript(script)
			if err != nil {
				return false, err
			}
			vm.scripts = append(vm.scripts, pops)
			// Set stack to be the stack from first script minus the
			// script itself
			vm.SetStack(vm.savedFirstStack[:len(vm.savedFirstStack)-1])
		} else {
			vm.scriptIdx ++
		}

		// there are zero length scripts in the wild
		if vm.scriptIdx < len(vm.scripts) && vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
			vm.scriptIdx++
		}
		vm.lastCodeSep = 0
		if vm.scriptIdx >= len(vm.scripts) {
			return true, nil
		}
	}

	return false, nil
}

func (vm *Engine) Execute() (err error) {
	done := false
	for !done {
		done, err = vm.Step()
		if err != nil {
			return err
		}
		var dstr, astr string
		// if we're tracing, dump the stacks.
		if vm.dstack.Depth() != 0 {
			dstr = "Stack:\n" + vm.dstack.String()
		}
		if vm.astack.Depth() != 0 {
			astr = "AltStack:\n" + vm.astack.String()
		}
		log.Info(dstr + astr)
	}
	return vm.CheckErrorCondition(true)
}

// executeOpcode peforms execution on the passed opcode.  It takes into account
// whether or not it is hidden by conditionals, but some rules still must be
// tested in this case.
func (vm *Engine) executeOpcode(pop *parsedOpcode) error {
	// Disabled opcodes are fail on program counter.
	if pop.isDisabled() {
		str := fmt.Sprintf("attempt to execute disabled opcode %s",
			pop.opcode.name)
		return errors.New(str)
	}

	// Always-illegal opcodes are fail on program counter.
	if pop.alwaysIllegal() {
		str := fmt.Sprintf("attempt to execute reserved opcode %s",
			pop.opcode.name)
		return errors.New(str)
	}

	// Note that this includes OP_RESERVED which counts as a push operation.
	if pop.opcode.value > OP_16 {
		vm.numOps++
		if vm.numOps > MaxOpsPerScript {
			str := fmt.Sprintf("exceeded max operation limit of %d",
				MaxOpsPerScript)
			return errors.New(str)
		}

	} else if len(pop.data) > MaxScriptElementSize {
		str := fmt.Sprintf("element size %d exceeds max allowed size %d",
			len(pop.data), MaxScriptElementSize)
		return errors.New(str)
	}

	// Nothing left to do when this is not a conditional opcode and it is
	// not in an executing branch.
	if !vm.isBranchExecuting() && !pop.isConditional() {
		return nil
	}

	// Ensure all executed data push opcodes use the minimal encoding when
	// the minimal data verification flag is set.
	if vm.dstack.verifyMinimalData && vm.isBranchExecuting() &&
		pop.opcode.value >= 0 && pop.opcode.value <= OP_PUSHDATA4 {

		if err := pop.checkMinimalDataPush(); err != nil {
			return err
		}
	}

	return pop.opcode.opfunc(pop, vm)
}

// validPC returns an error if the current script position is valid for
// execution, nil otherwise.
func (vm *Engine) validPC() error {
	if vm.scriptIdx >= len(vm.scripts) {
		str := fmt.Sprintf("past input scripts %v:%v %v:xxxx",
			vm.scriptIdx, vm.scriptOff, len(vm.scripts))
		return errors.New(str)
	}
	if vm.scriptOff >= len(vm.scripts[vm.scriptIdx]) {
		str := fmt.Sprintf("past input scripts %v:%v %v:%04d",
			vm.scriptIdx, vm.scriptOff, vm.scriptIdx,
			len(vm.scripts[vm.scriptIdx]))
		return errors.New(str)
	}
	return nil
}
