#pragma once

#include <llvm/MC/MCInst.h>
#include <llvm/Object/ObjectFile.h>
#include <llvm/TargetParser/Triple.h>

#include <library/cpp/logger/log.h>

#include <util/generic/array_ref.h>
#include <util/generic/function_ref.h>
#include <util/generic/maybe.h>
#include <util/generic/hash.h>
#include <util/generic/vector.h>

#include <expected>
#include <variant>


namespace NPerforator::NAsm::NX86 {

enum class EDecodeInstructionError {
    NoError,
    TargetLookupFailed,
    SubtargetInfoCreationFailed,
    DisassemblyFailed,
};

// Note: it is caller responsibility to return false from instCallback to stop decoding instructions
EDecodeInstructionError DecodeInstructions(
    TLog& log,
    const llvm::Triple& triple,
    TConstArrayRef<ui8> bytecode,
    TFunctionRef<bool(const llvm::MCInst&, ui64 size)> instCallback
);

struct TImmediateValue {
    i64 Value;
};

struct TMemoryValue {
    i64 Address;
};

using TRegisterValue = std::variant<TImmediateValue, TMemoryValue>;

struct TState {
    THashMap<unsigned int, TRegisterValue> StaticRegisterValues;

    void SetImmediate(unsigned int reg, i64 value);
    void SetMemoryRef(unsigned int reg, i64 address);
    TMaybe<i64> GetImmediateValue(unsigned int reg) const;
    TMaybe<i64> GetMemoryAddress(unsigned int reg) const;
    bool HasKnownValue(unsigned int reg) const;
};

TState MakeInitialState(ui64 initialRIP);

TMaybe<ui64> GetRegisterValueOrAddress(const TState& state, unsigned int reg);

using TEvaluationStopCondition = TFunctionRef<bool(const TState&, const llvm::MCInst&)>;

bool IsJump(const llvm::MCInst& inst);
bool IsCall(const llvm::MCInst& inst);
bool IsRet(const llvm::MCInst& inst);
bool IsPassControlFlow(const llvm::MCInst& inst);

TEvaluationStopCondition MakeStopOnPassControlFlowCondition();
TEvaluationStopCondition MakeStopOnCallCondition();
TEvaluationStopCondition MakeStopOnRetCondition();


class IInstructionEvaluator {
public:
    virtual void Evaluate(TState& state, const llvm::MCInst& inst) = 0;

    virtual ~IInstructionEvaluator() = default;
};

THolder<IInstructionEvaluator> MakeDefaultInstructionEvaluator();


class TBytecodeEvaluator {
public:
    TBytecodeEvaluator(
        const llvm::Triple& triple,
        TState initialState,
        TConstArrayRef<ui8> bytecode,
        IInstructionEvaluator& instructionEvaluator,
        TEvaluationStopCondition stopCondition)
        : Triple_(triple)
        , InitialState_(std::move(initialState))
        , Bytecode_(bytecode)
        , InstructionEvaluator_(instructionEvaluator)
        , StopCondition_(stopCondition)
    {
    }

    struct TResult {
        TState State;
        bool StoppedOnCondition;
    };

    std::expected<TResult, EDecodeInstructionError> Evaluate();

private:
    llvm::Triple Triple_;
    TState InitialState_;
    TConstArrayRef<ui8> Bytecode_;
    IInstructionEvaluator& InstructionEvaluator_;
    TEvaluationStopCondition StopCondition_;
};

} // namespace NPerforator::NAsm::NX86
