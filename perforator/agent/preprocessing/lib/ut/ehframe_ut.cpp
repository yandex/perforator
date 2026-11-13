#include <perforator/agent/preprocessing/lib/ehframe.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <llvm/Support/DataExtractor.h>

#include <vector>

using namespace NPerforator::NBinaryProcessing::NUnwind;

namespace {

// LEB128 operands: SLEB128(-8) = 0x78, SLEB128(8) = ULEB128(8) = 0x08,
// SLEB128(152) = 0x98 0x01, SLEB128(160) = 0xa0 0x01.
constexpr ui8 OP_breg0 = llvm::dwarf::DW_OP_breg0;
constexpr ui8 OP_breg6 = llvm::dwarf::DW_OP_breg6;
constexpr ui8 OP_breg7 = llvm::dwarf::DW_OP_breg7;
constexpr ui8 OP_deref = llvm::dwarf::DW_OP_deref;
constexpr ui8 OP_plus_uconst = llvm::dwarf::DW_OP_plus_uconst;
constexpr ui8 OP_lit0 = llvm::dwarf::DW_OP_lit0;

UnwindRule ConvertCfaExpression(const std::vector<ui8>& bytes) {
    llvm::DataExtractor data{llvm::ArrayRef<ui8>{bytes}, /*IsLittleEndian=*/true, /*AddressSize=*/8};
    llvm::DWARFExpression expr{data, /*AddressSize=*/8};

    UnwindRule rule;
    FillRegisterLocation(&rule, llvm::dwarf::UnwindLocation::createIsDWARFExpression(expr));
    return rule;
}

// A register saved at the address computed by the expression (DW_CFA_expression).
UnwindRule ConvertRegisterExpression(const std::vector<ui8>& bytes) {
    llvm::DataExtractor data{llvm::ArrayRef<ui8>{bytes}, /*IsLittleEndian=*/true, /*AddressSize=*/8};
    llvm::DWARFExpression expr{data, /*AddressSize=*/8};

    UnwindRule rule;
    FillRegisterLocation(&rule, llvm::dwarf::UnwindLocation::createAtDWARFExpression(expr));
    return rule;
}

void ExpectRegisterDerefOffset(const UnwindRule& rule, ui32 regno, i32 offset, i32 bias) {
    ASSERT_TRUE(rule.has_register_deref_offset()) << rule.DebugString();
    EXPECT_EQ(rule.register_deref_offset().register_(), regno);
    EXPECT_EQ(rule.register_deref_offset().offset(), offset);
    EXPECT_EQ(rule.register_deref_offset().bias(), bias);
}

} // namespace

TEST(EhFrameCfaExpression, DerefPlus8OpenSSL) {
    // openssl sha512_block_data_order_avx2: DW_OP_breg7 (rsp): 152; DW_OP_deref; DW_OP_plus_uconst: 8
    auto rule = ConvertCfaExpression({OP_breg7, 0x98, 0x01, OP_deref, OP_plus_uconst, 0x08});
    ExpectRegisterDerefOffset(rule, 7, 152, 8);
}

TEST(EhFrameCfaExpression, DerefPlus8NegativeOffset) {
    // DW_OP_breg7 (rsp): -8; DW_OP_deref; DW_OP_plus_uconst: 8
    auto rule = ConvertCfaExpression({OP_breg7, 0x78, OP_deref, OP_plus_uconst, 0x08});
    ExpectRegisterDerefOffset(rule, 7, -8, 8);
}

TEST(EhFrameCfaExpression, DerefGccDrap) {
    // gcc stack realignment through DRAP: DW_OP_breg6 (rbp): -8; DW_OP_deref
    auto rule = ConvertCfaExpression({OP_breg6, 0x78, OP_deref});
    ExpectRegisterDerefOffset(rule, 6, -8, 0);
}

TEST(EhFrameCfaExpression, DerefStackPointerNoBias) {
    // glibc: DW_OP_breg7 (rsp): 160; DW_OP_deref
    auto rule = ConvertCfaExpression({OP_breg7, 0xa0, 0x01, OP_deref});
    ExpectRegisterDerefOffset(rule, 7, 160, 0);
}

TEST(EhFrameCfaExpression, NotDeref) {
    // Truncated or extended forms and other base registers are not deref rules.
    EXPECT_FALSE(ConvertCfaExpression({OP_breg6, 0x78}).has_register_deref_offset());
    EXPECT_FALSE(ConvertCfaExpression({OP_breg6, 0x78, OP_deref, OP_plus_uconst, 0x08, OP_lit0}).has_register_deref_offset());
    EXPECT_FALSE(ConvertCfaExpression({OP_breg0, 0x08, OP_deref}).has_register_deref_offset());
}

TEST(EhFrameRegisterExpression, RbpSavedAtRbp) {
    // gcc stack realignment through DRAP: DW_CFA_expression: r6 (rbp) (DW_OP_breg6 (rbp): 0)
    auto rule = ConvertRegisterExpression({OP_breg6, 0x00});
    ASSERT_TRUE(rule.has_register_offset()) << rule.DebugString();
    EXPECT_EQ(rule.register_offset().register_(), 6u);
    EXPECT_EQ(rule.register_offset().offset(), 0);
    EXPECT_TRUE(rule.dereference());
}

TEST(EhFrameCfaExpression, IndexedDerefPlus8OpenSSL) {
    // openssl bn_mul_mont: DW_OP_breg7 (rsp): 8; DW_OP_breg9 (r9): 0; DW_OP_lit8; DW_OP_mul; DW_OP_plus;
    // DW_OP_deref; DW_OP_plus_uconst: 8
    constexpr ui8 OP_breg9 = llvm::dwarf::DW_OP_breg9;
    constexpr ui8 OP_lit8 = llvm::dwarf::DW_OP_lit8;
    constexpr ui8 OP_mul = llvm::dwarf::DW_OP_mul;
    constexpr ui8 OP_plus = llvm::dwarf::DW_OP_plus;

    auto rule = ConvertCfaExpression({OP_breg7, 0x08, OP_breg9, 0x00, OP_lit8, OP_mul, OP_plus, OP_deref, OP_plus_uconst, 0x08});
    ASSERT_TRUE(rule.has_register_indexed_deref_offset()) << rule.DebugString();
    const auto& deref = rule.register_indexed_deref_offset();
    EXPECT_EQ(deref.register_(), 7u);
    EXPECT_EQ(deref.offset(), 8);
    EXPECT_EQ(deref.index_register(), 9u);
    EXPECT_EQ(deref.scale(), 8u);
    EXPECT_EQ(deref.bias(), 8);

    // A different scale is not this rule.
    constexpr ui8 OP_lit4 = llvm::dwarf::DW_OP_lit4;
    EXPECT_FALSE(ConvertCfaExpression({OP_breg7, 0x08, OP_breg9, 0x00, OP_lit4, OP_mul, OP_plus, OP_deref, OP_plus_uconst, 0x08})
        .has_register_indexed_deref_offset());
}
