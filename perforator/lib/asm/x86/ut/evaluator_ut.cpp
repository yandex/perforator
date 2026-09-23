#include <perforator/lib/asm/evaluator.h>

#ifdef __GNUC__
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#endif
#include <contrib/libs/llvm22/lib/Target/X86/X86InstrInfo.h>
#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif

#include <llvm/Support/TargetSelect.h>
#include <llvm/MC/MCInst.h>
#include <llvm/MC/MCRegisterInfo.h>
#include <llvm/Object/ObjectFile.h>
#include <llvm/MC/MCInstBuilder.h>

#include <library/cpp/testing/gtest/gtest.h>
#include <library/cpp/logger/global/global.h>

#include <util/generic/vector.h>
#include <util/string/builder.h>

using namespace NPerforator::NAsm;

class EvaluatorTest : public ::testing::Test {
protected:
    void SetUp() override {}

    void TearDown() override {}
};

TEST_F(EvaluatorTest, StateBasics) {
    TState state;
    const unsigned int testReg1 = 123;
    const unsigned int testReg2 = 456;

    EXPECT_FALSE(state.HasKnownValue(testReg1));
    EXPECT_FALSE(state.GetImmediateValue(testReg1).Defined());
    EXPECT_FALSE(state.GetMemoryAddress(testReg1).Defined());

    const i64 immediateValue = 0x12345678;
    state.SetImmediate(testReg1, immediateValue);
    EXPECT_TRUE(state.HasKnownValue(testReg1));
    EXPECT_TRUE(state.GetImmediateValue(testReg1).Defined());
    EXPECT_EQ(*state.GetImmediateValue(testReg1), immediateValue);
    EXPECT_FALSE(state.GetMemoryAddress(testReg1).Defined());

    const i64 memoryAddress = 0xABCDEF;
    state.SetMemoryRef(testReg2, memoryAddress);
    EXPECT_TRUE(state.HasKnownValue(testReg2));
    EXPECT_FALSE(state.GetImmediateValue(testReg2).Defined());
    EXPECT_TRUE(state.GetMemoryAddress(testReg2).Defined());
    EXPECT_EQ(*state.GetMemoryAddress(testReg2), memoryAddress);

    state.SetImmediate(testReg2, immediateValue);
    EXPECT_TRUE(state.GetImmediateValue(testReg2).Defined());
    EXPECT_FALSE(state.GetMemoryAddress(testReg2).Defined());
    EXPECT_EQ(*state.GetImmediateValue(testReg2), immediateValue);
}

TEST_F(EvaluatorTest, StopConditionFunctions) {
    const unsigned int initialRIP = 0x1000;
    TState initialState = MakeInitialState(initialRIP);

    auto cfStopCondition = MakeStopOnPassControlFlowCondition();

    llvm::MCInst callInst = llvm::MCInstBuilder(llvm::X86::CALLpcrel32).addImm(0);

    EXPECT_TRUE(IsJump(callInst) || IsCall(callInst));
    EXPECT_TRUE(cfStopCondition(initialState, callInst));

    llvm::MCInst movInst = llvm::MCInstBuilder(llvm::X86::MOV32ri).addReg(llvm::X86::EAX).addImm(1);
    EXPECT_FALSE(IsJump(movInst) || IsCall(movInst));
    EXPECT_FALSE(cfStopCondition(initialState, movInst));
}

TEST_F(EvaluatorTest, BytecodeEvaluator) {
    const unsigned int initialRIP = 0x1000;
    TState initialState = MakeInitialState(initialRIP);

    auto defaultEvaluator = MakeDefaultInstructionEvaluator();
    EXPECT_NE(defaultEvaluator, nullptr);

    // Using a sequence of x86 instructions with both MOV and LEA:
    // 1. MOV EAX, 42              (B8 2A 00 00 00)
    // 2. MOV EBX, 10              (BB 0A 00 00 00)
    // 3. LEA ECX, [EAX+10]        (8D 48 0A)
    // 4. LEA EDX, [ECX+EBX*2]     (8D 54 59 00)

    TVector<ui8> bytecode = {
        0xB8, 0x2A, 0x00, 0x00, 0x00,  // MOV EAX, 42
        0xBB, 0x0A, 0x00, 0x00, 0x00,  // MOV EBX, 10
        0x8D, 0x48, 0x0A,              // LEA ECX, [EAX+10]
        0x8D, 0x54, 0x59, 0x00         // LEA EDX, [ECX+EBX*2]
    };

    llvm::Triple triple("x86_64-unknown-linux-gnu");
    auto cfStopCondition = MakeStopOnPassControlFlowCondition();

    TBytecodeEvaluator evaluator(
        triple,
        initialState,
        TConstArrayRef<ui8>(bytecode.data(), bytecode.size()),
        *defaultEvaluator,
        cfStopCondition
    );

    auto result = evaluator.Evaluate();
    EXPECT_TRUE(result.has_value());

    auto raxValue = result->State.GetImmediateValue(llvm::X86::RAX);
    EXPECT_TRUE(raxValue.Defined());
    EXPECT_EQ(*raxValue, 42);

    auto rbxValue = result->State.GetImmediateValue(llvm::X86::RBX);
    EXPECT_TRUE(rbxValue.Defined());
    EXPECT_EQ(*rbxValue, 10);

    auto rcxValue = result->State.GetImmediateValue(llvm::X86::RCX);
    EXPECT_TRUE(rcxValue.Defined());
    EXPECT_EQ(*rcxValue, 52);  // RAX(42) + 10 = 52, upper 32 bits zero

    auto rdxValue = result->State.GetImmediateValue(llvm::X86::RDX);
    EXPECT_TRUE(rdxValue.Defined());
    EXPECT_EQ(*rdxValue, 72);  // RCX(52) + RBX(10)*2 = 72, upper 32 bits zero

    auto ripValue = result->State.GetImmediateValue(llvm::X86::RIP);
    EXPECT_TRUE(ripValue.Defined());
    EXPECT_EQ(*ripValue, initialRIP + static_cast<unsigned int>(bytecode.size()));
}

TEST_F(EvaluatorTest, DefaultEvaluator) {
    const unsigned int initialRIP = 0x1000;
    TState initialState = MakeInitialState(initialRIP);

    auto defaultEvaluator = MakeDefaultInstructionEvaluator();
    EXPECT_NE(defaultEvaluator, nullptr);

    TState testState = initialState;

    llvm::MCInst movRax1 = llvm::MCInstBuilder(llvm::X86::MOV32ri)
        .addReg(llvm::X86::EAX)
        .addImm(1);
    defaultEvaluator->Evaluate(testState, movRax1);

    auto eaxValue = testState.GetImmediateValue(llvm::X86::EAX);
    EXPECT_TRUE(eaxValue.Defined());
    EXPECT_EQ(*eaxValue, 1);

    auto raxValue = testState.GetImmediateValue(llvm::X86::RAX);
    EXPECT_TRUE(raxValue.Defined());
    EXPECT_EQ(*raxValue, 1);

    llvm::MCInst leaInst = llvm::MCInstBuilder(llvm::X86::LEA32r)
        .addReg(llvm::X86::EBX)
        .addReg(llvm::X86::RAX)
        .addImm(1)
        .addReg(0)
        .addImm(0x100)
        .addReg(0);

    defaultEvaluator->Evaluate(testState, leaInst);

    auto ebxValue = testState.GetImmediateValue(llvm::X86::EBX);
    EXPECT_TRUE(ebxValue.Defined());
    EXPECT_EQ(*ebxValue, 0x101);

    auto rbxValue = testState.GetImmediateValue(llvm::X86::RBX);
    EXPECT_TRUE(rbxValue.Defined());
    EXPECT_EQ(*rbxValue, 0x101);

    llvm::MCInst movRcxRax = llvm::MCInstBuilder(llvm::X86::MOV32rr)
        .addReg(llvm::X86::ECX)
        .addReg(llvm::X86::EAX);

    defaultEvaluator->Evaluate(testState, movRcxRax);

    auto ecxValue = testState.GetImmediateValue(llvm::X86::ECX);
    EXPECT_TRUE(ecxValue.Defined());
    EXPECT_EQ(*ecxValue, 1);

    auto rcxValue = testState.GetImmediateValue(llvm::X86::RCX);
    EXPECT_TRUE(rcxValue.Defined());
    EXPECT_EQ(*rcxValue, 1);

    llvm::MCInst lea64_32Inst = llvm::MCInstBuilder(llvm::X86::LEA64_32r)
        .addReg(llvm::X86::EAX)
        .addReg(llvm::X86::RBX)
        .addImm(1)
        .addReg(0)
        .addImm(0x100000000)
        .addReg(0);

    TState testState64_32 = initialState;
    testState64_32.SetImmediate(llvm::X86::RBX, 0x200000000);
    defaultEvaluator->Evaluate(testState64_32, lea64_32Inst);

    auto eaxValue64_32 = testState64_32.GetImmediateValue(llvm::X86::EAX);
    EXPECT_TRUE(eaxValue64_32.Defined());
    EXPECT_EQ(*eaxValue64_32, 0);

    auto raxValue64_32 = testState64_32.GetImmediateValue(llvm::X86::RAX);
    EXPECT_TRUE(raxValue64_32.Defined());
    EXPECT_EQ(*raxValue64_32, 0);
}

