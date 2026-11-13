#include <perforator/agent/preprocessing/lib/dwarf_matcher.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <llvm/Support/DataExtractor.h>

#include <vector>

using namespace NPerforator::NBinaryProcessing::NUnwind;

namespace {

// Operands below are single-byte LEB128: SLEB128(8) = 0x08, SLEB128(-8) = 0x78, ULEB128(8) = 0x08.
constexpr ui8 OP_breg6 = llvm::dwarf::DW_OP_breg6;
constexpr ui8 OP_breg7 = llvm::dwarf::DW_OP_breg7;
constexpr ui8 OP_deref = llvm::dwarf::DW_OP_deref;
constexpr ui8 OP_plus_uconst = llvm::dwarf::DW_OP_plus_uconst;

bool Matches(TDwarfExpressionPattern pattern, const std::vector<ui8>& bytes) {
    llvm::DataExtractor data{llvm::ArrayRef<ui8>{bytes}, /*IsLittleEndian=*/true, /*AddressSize=*/8};
    llvm::DWARFExpression expr{data, /*AddressSize=*/8};
    return pattern.Matches(expr);
}

} // namespace

TEST(DwarfExpressionPattern, ExactMatch) {
    ui64 offset = 0;
    ui64 bias = 0;
    auto pattern = TDwarfExpressionPattern{}
        .Push(llvm::dwarf::DW_OP_breg6, Wildcard(&offset))
        .Push(llvm::dwarf::DW_OP_deref)
        .Push(llvm::dwarf::DW_OP_plus_uconst, Wildcard(&bias));

    EXPECT_TRUE(Matches(pattern, {OP_breg6, 0x78, OP_deref, OP_plus_uconst, 0x08}));
    EXPECT_EQ(static_cast<i64>(offset), -8);
    EXPECT_EQ(bias, 8u);
}

TEST(DwarfExpressionPattern, OperandMismatch) {
    auto pattern = TDwarfExpressionPattern{}
        .Push(llvm::dwarf::DW_OP_breg7, 8)
        .Push(llvm::dwarf::DW_OP_deref);

    EXPECT_FALSE(Matches(pattern, {OP_breg7, 0x78, OP_deref}));
}

TEST(DwarfExpressionPattern, ExpressionLongerThanPattern) {
    auto pattern = TDwarfExpressionPattern{}
        .Push(llvm::dwarf::DW_OP_breg7, 8)
        .Push(llvm::dwarf::DW_OP_deref);

    EXPECT_FALSE(Matches(pattern, {OP_breg7, 0x08, OP_deref, OP_plus_uconst, 0x08}));
}

TEST(DwarfExpressionPattern, ExpressionShorterThanPattern) {
    auto pattern = TDwarfExpressionPattern{}
        .Push(llvm::dwarf::DW_OP_breg6, static_cast<ui64>(-8))
        .Push(llvm::dwarf::DW_OP_deref)
        .Push(llvm::dwarf::DW_OP_plus_uconst, 8);

    EXPECT_FALSE(Matches(pattern, {OP_breg6, 0x78, OP_deref}));
    EXPECT_FALSE(Matches(pattern, {OP_breg6, 0x78}));
}

TEST(DwarfExpressionPattern, PatternRepeatingLastOperation) {
    // Past the end, the expression iterator still holds the last operation.
    // A pattern that repeats it must not match a shorter expression.
    auto pattern = TDwarfExpressionPattern{}
        .Push(llvm::dwarf::DW_OP_breg7, 8)
        .Push(llvm::dwarf::DW_OP_deref)
        .Push(llvm::dwarf::DW_OP_deref);

    EXPECT_FALSE(Matches(pattern, {OP_breg7, 0x08, OP_deref}));
    EXPECT_TRUE(Matches(pattern, {OP_breg7, 0x08, OP_deref, OP_deref}));
}

TEST(DwarfExpressionPattern, EmptyExpression) {
    auto pattern = TDwarfExpressionPattern{}.Push(llvm::dwarf::DW_OP_deref);

    EXPECT_FALSE(Matches(pattern, {}));
}
