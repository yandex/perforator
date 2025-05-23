#include "evaluator.h"

#ifdef __GNUC__
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#endif
#include <contrib/libs/llvm18/lib/Target/X86/X86InstrInfo.h>
#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif

#include <llvm/Support/TargetSelect.h>
#include <llvm/MC/MCDisassembler/MCDisassembler.h>
#include <llvm/MC/MCContext.h>
#include <llvm/MC/MCInst.h>
#include <llvm/MC/MCInstPrinter.h>
#include <llvm/MC/MCRegisterInfo.h>
#include <llvm/MC/MCSubtargetInfo.h>
#include <llvm/MC/MCAsmInfo.h>
#include <llvm/Support/MemoryBuffer.h>
#include <llvm/Support/SourceMgr.h>
#include <llvm/Support/raw_ostream.h>
#include <llvm/Target/TargetMachine.h>
#include <llvm/MC/MCInstBuilder.h>
#include <llvm/MC/MCObjectFileInfo.h>
#include <llvm/MC/TargetRegistry.h>
#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>

#include <library/cpp/logger/global/global.h>
#include <util/generic/function_ref.h>

namespace NPerforator::NAsm::NX86 {

EDecodeInstructionError DecodeInstructions(
    TLog& log,
    const llvm::Triple& triple,
    TConstArrayRef<ui8> bytecode,
    TFunctionRef<bool(const llvm::MCInst&, ui64 size)> instCallback
) {
    LLVMInitializeX86Target();
    LLVMInitializeX86TargetInfo();
    LLVMInitializeX86TargetMC();
    LLVMInitializeX86Disassembler();

    std::string error;
    const llvm::Target* target = llvm::TargetRegistry::lookupTarget(triple.getTriple(), error);
    if (!target) {
        log << TLOG_ERR << "Failed to lookup target by triple " << triple.getTriple() << ' ' << error << Endl;
        return EDecodeInstructionError::TargetLookupFailed;
    }

    THolder<llvm::MCRegisterInfo> mri(target->createMCRegInfo(triple.getTriple()));

    llvm::MCTargetOptions options;
    THolder<llvm::MCAsmInfo> asmInfo(
        target->createMCAsmInfo(
            *mri,
            triple.getTriple(),
            options
        )
    );

    THolder<llvm::MCSubtargetInfo> subTargetInfo(
        target->createMCSubtargetInfo(triple.getTriple(), "", "")
    );
    if (subTargetInfo == nullptr) {
        return EDecodeInstructionError::SubtargetInfoCreationFailed;
    }

    llvm::MCContext context(triple, asmInfo.get(), mri.get(), subTargetInfo.get());

    THolder<llvm::MCDisassembler> disasm(target->createMCDisassembler(*subTargetInfo, context));

    for (size_t i = 0; i < bytecode.size(); ) {
        llvm::MCInst inst;
        ui64 size = 0;
        auto status = disasm->getInstruction(inst, size, llvm::ArrayRef<ui8>(bytecode.data() + i, bytecode.size() - i), i, llvm::nulls());

        if (status != llvm::MCDisassembler::Success) {
            return EDecodeInstructionError::DisassemblyFailed;
        }

        if (!instCallback(inst, size)) {
            return EDecodeInstructionError::NoError;
        }

        // Move to the next instruction
        i += size;
    }

    return EDecodeInstructionError::NoError;
}

bool IsCall(const llvm::MCInst& inst) {
    return inst.getOpcode() == llvm::X86::CALL64pcrel32 ||
           inst.getOpcode() == llvm::X86::CALL64r ||
           inst.getOpcode() == llvm::X86::CALL64m ||
           inst.getOpcode() == llvm::X86::CALLpcrel32;
}

bool IsJump(const llvm::MCInst& inst) {
    return inst.getOpcode() == llvm::X86::JMP64r ||
           inst.getOpcode() == llvm::X86::JMP64m ||
           inst.getOpcode() == llvm::X86::JMP32r ||
           inst.getOpcode() == llvm::X86::JMP32m;
}

bool IsRet(const llvm::MCInst& inst) {
    return llvm::X86::isRET(inst.getOpcode()) ||
            llvm::X86::isRETFQ(inst.getOpcode()) ||
            llvm::X86::isRETF(inst.getOpcode());
}

bool IsPassControlFlow(const llvm::MCInst& inst) {
    return IsJump(inst) || IsCall(inst) || IsRet(inst);
}

TEvaluationStopCondition MakeStopOnPassControlFlowCondition() {
    return [](const TState&, const llvm::MCInst& inst) -> bool {
        return IsPassControlFlow(inst);
    };
}

TEvaluationStopCondition MakeStopOnCallCondition() {
    return [](const TState&, const llvm::MCInst& inst) -> bool {
        return IsCall(inst);
    };
}

TEvaluationStopCondition MakeStopOnRetCondition() {
    return [](const TState&, const llvm::MCInst& inst) -> bool {
        return IsRet(inst);
    };
}

TState MakeInitialState(ui64 initialRIP) {
    TState state;
    state.SetImmediate(llvm::X86::RIP, initialRIP);
    return state;
}

TMaybe<ui64> GetRegisterValueOrAddress(const TState& state, unsigned int reg) {
    if (!state.HasKnownValue(reg)) {
        return Nothing();
    }

    auto immValue = state.GetImmediateValue(reg);
    if (immValue) {
        return static_cast<ui64>(*immValue);
    }

    auto memAddr = state.GetMemoryAddress(reg);
    if (memAddr) {
        return static_cast<ui64>(*memAddr);
    }

    return Nothing();
}

unsigned int GetBaseRegister(unsigned int reg) {
    switch (reg) {
        case llvm::X86::EIP:
        case llvm::X86::RIP:
            return llvm::X86::RIP;

        case llvm::X86::EAX:
            return llvm::X86::RAX;

        case llvm::X86::EBX:
            return llvm::X86::RBX;

        case llvm::X86::ECX:
            return llvm::X86::RCX;

        case llvm::X86::EDX:
            return llvm::X86::RDX;

        case llvm::X86::ESI:
            return llvm::X86::RSI;

        case llvm::X86::EDI:
            return llvm::X86::RDI;

        case llvm::X86::ESP:
            return llvm::X86::RSP;

        case llvm::X86::EBP:
            return llvm::X86::RBP;

        default:
            return reg;
    }
}

i64 MaskValueForRegister(unsigned int reg, i64 value) {
    // Only handle 32-bit registers, 64-bit registers don't need masking
    if (reg == llvm::X86::EAX || reg == llvm::X86::EBX ||
        reg == llvm::X86::ECX || reg == llvm::X86::EDX ||
        reg == llvm::X86::ESI || reg == llvm::X86::EDI ||
        reg == llvm::X86::EBP || reg == llvm::X86::ESP ||
        reg == llvm::X86::EIP) {
        return value & 0xFFFFFFFF;
    }

    return value;
}

void TState::SetImmediate(unsigned int reg, i64 value) {
    StaticRegisterValues[GetBaseRegister(reg)] = TImmediateValue{MaskValueForRegister(reg, value)};
}

void TState::SetMemoryRef(unsigned int reg, i64 address) {
    StaticRegisterValues[GetBaseRegister(reg)] = TMemoryValue{address};
}

TMaybe<i64> TState::GetImmediateValue(unsigned int reg) const {
    auto it = StaticRegisterValues.find(GetBaseRegister(reg));
    if (it != StaticRegisterValues.end() && std::holds_alternative<TImmediateValue>(it->second)) {
        return std::get<TImmediateValue>(it->second).Value;
    }
    return Nothing();
}

TMaybe<i64> TState::GetMemoryAddress(unsigned int reg) const {
    auto it = StaticRegisterValues.find(GetBaseRegister(reg));
    if (it != StaticRegisterValues.end() && std::holds_alternative<TMemoryValue>(it->second)) {
        return std::get<TMemoryValue>(it->second).Address;
    }
    return Nothing();
}

bool TState::HasKnownValue(unsigned int reg) const {
    return StaticRegisterValues.contains(GetBaseRegister(reg));
}

class TMovEvaluator : public IInstructionEvaluator {
public:
    void Evaluate(TState& state, const llvm::MCInst& inst) override {
        unsigned int opcode = inst.getOpcode();
        switch (opcode) {
            // MOV register to register
            case llvm::X86::MOV64rr:
            case llvm::X86::MOV32rr: {
                if (inst.getNumOperands() >= 2 &&
                    inst.getOperand(0).isReg() &&
                    inst.getOperand(1).isReg()) {

                    unsigned int dstReg = inst.getOperand(0).getReg();
                    unsigned int srcReg = inst.getOperand(1).getReg();

                    if (state.HasKnownValue(srcReg)) {
                        auto value = state.GetImmediateValue(srcReg);
                        if (value) {
                            state.SetImmediate(dstReg, *value);
                        }
                    } else {
                        state.StaticRegisterValues.erase(GetBaseRegister(dstReg));
                    }
                }
                break;
            }

            // MOV immediate to register
            case llvm::X86::MOV64ri:
            case llvm::X86::MOV32ri: {
                if (inst.getNumOperands() >= 2 &&
                    inst.getOperand(0).isReg() &&
                    inst.getOperand(1).isImm()) {

                    unsigned int dstReg = inst.getOperand(0).getReg();
                    i64 immValue = inst.getOperand(1).getImm();
                    state.SetImmediate(dstReg, immValue);
                }
                break;
            }

            // Memory access MOV instructions
            case llvm::X86::MOV64rm:
            case llvm::X86::MOV32rm: {
                // Format: MOV dst, [base + scale*index + disp]
                if (inst.getNumOperands() >= 6 && inst.getOperand(0).isReg()) {
                    unsigned int dstReg = inst.getOperand(0).getReg();
                    unsigned int baseReg = inst.getOperand(1).getReg();
                    unsigned int scaleValue = inst.getOperand(2).getImm();
                    unsigned int indexReg = inst.getOperand(3).getReg();
                    i64 dispValue = inst.getOperand(4).getImm();

                    bool canComputeAddress = true;
                    i64 computedAddr = dispValue;

                    if (baseReg != llvm::X86::NoRegister) {
                        auto baseValue = state.GetImmediateValue(baseReg);
                        if (baseValue) {
                            computedAddr += *baseValue;
                        } else {
                            canComputeAddress = false;
                        }
                    }

                    if (indexReg != llvm::X86::NoRegister) {
                        auto indexValue = state.GetImmediateValue(indexReg);
                        if (indexValue) {
                            computedAddr += scaleValue * (*indexValue);
                        } else {
                            canComputeAddress = false;
                        }
                    }

                    if (canComputeAddress) {
                        state.SetMemoryRef(dstReg, computedAddr);
                    } else {
                        state.StaticRegisterValues.erase(dstReg);
                    }
                }
                break;
            }
        }
    }
};

class TLeaEvaluator : public IInstructionEvaluator {
public:
    void Evaluate(TState& state, const llvm::MCInst& inst) override {
        auto opcode = inst.getOpcode();

        if (opcode == llvm::X86::LEA64r || opcode == llvm::X86::LEA32r || opcode == llvm::X86::LEA64_32r) {
            unsigned int numOperands = inst.getNumOperands();

            // Standard form with 6 operands
            if (numOperands >= 6 && inst.getOperand(0).isReg()) {
                unsigned int dstReg = inst.getOperand(0).getReg();
                unsigned int baseReg = inst.getOperand(1).getReg();
                unsigned int scaleValue = inst.getOperand(2).getImm();
                unsigned int indexReg = inst.getOperand(3).getReg();
                i64 dispValue = inst.getOperand(4).getImm();

                bool canCompute = true;
                i64 computedAddr = dispValue;

                if (baseReg != 0) {
                    auto baseValue = state.GetImmediateValue(baseReg);
                    if (baseValue) {
                        computedAddr += *baseValue;
                    } else {
                        canCompute = false;
                    }
                }

                if (indexReg != 0) {
                    auto baseValue = state.GetImmediateValue(indexReg);
                    if (baseValue) {
                        computedAddr += scaleValue * (*baseValue);
                    } else {
                        canCompute = false;
                    }
                }

                if (canCompute && opcode == llvm::X86::LEA64_32r) {
                    computedAddr &= 0xFFFFFFFF;
                }

                if (canCompute) {
                    state.SetImmediate(dstReg, computedAddr);
                } else {
                    state.StaticRegisterValues.erase(GetBaseRegister(dstReg));
                }
            }
            // If format doesn't match expected, clear the value
            else if (numOperands > 0 && inst.getOperand(0).isReg()) {
                unsigned int dstReg = inst.getOperand(0).getReg();
                state.StaticRegisterValues.erase(GetBaseRegister(dstReg));
            }
        }
    }
};

class TInstructionEvaluator : public IInstructionEvaluator {
public:
    TInstructionEvaluator() {
        Evaluators_[llvm::X86::MOV64rr] = MakeHolder<TMovEvaluator>();
        Evaluators_[llvm::X86::MOV32rr] = MakeHolder<TMovEvaluator>();
        Evaluators_[llvm::X86::MOV64ri] = MakeHolder<TMovEvaluator>();
        Evaluators_[llvm::X86::MOV32ri] = MakeHolder<TMovEvaluator>();
        Evaluators_[llvm::X86::MOV64rm] = MakeHolder<TMovEvaluator>();
        Evaluators_[llvm::X86::MOV32rm] = MakeHolder<TMovEvaluator>();
        Evaluators_[llvm::X86::LEA64r] = MakeHolder<TLeaEvaluator>();
        Evaluators_[llvm::X86::LEA32r] = MakeHolder<TLeaEvaluator>();
        Evaluators_[llvm::X86::LEA64_32r] = MakeHolder<TLeaEvaluator>();
    }

    void Evaluate(TState& state, const llvm::MCInst& inst) {
        auto opcode = inst.getOpcode();
        if (Evaluators_.contains(opcode)) {
            Evaluators_[opcode]->Evaluate(state, inst);
        }
    }

private:
    THashMap<unsigned int, THolder<IInstructionEvaluator>> Evaluators_;
};

THolder<IInstructionEvaluator> MakeDefaultInstructionEvaluator() {
    return MakeHolder<TInstructionEvaluator>();
}

std::expected<TBytecodeEvaluator::TResult, EDecodeInstructionError> TBytecodeEvaluator::Evaluate() {
    TBytecodeEvaluator::TResult result;
    result.State = InitialState_;

    auto decodeError = DecodeInstructions(TLoggerOperator<TGlobalLog>::Log(), Triple_, Bytecode_, [&](const llvm::MCInst& inst, ui64 size) {
        auto currentRIP = result.State.GetImmediateValue(llvm::X86::RIP);
        if (currentRIP) {
            result.State.SetImmediate(llvm::X86::RIP, *currentRIP + size);
        }

        if (StopCondition_(result.State, inst)) {
            result.StoppedOnCondition = true;
            return false;
        }

        InstructionEvaluator_.Evaluate(result.State, inst);
        return true;
    });

    if (decodeError != EDecodeInstructionError::NoError) {
        return std::unexpected(decodeError);
    }

    return result;
}

} // namespace NPerforator::NAsm::NX86
