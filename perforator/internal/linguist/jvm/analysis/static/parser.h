#pragma once

#include <contrib/libs/llvm18/include/llvm/DebugInfo/DWARF/DWARFDie.h>

#include <util/string/join.h>
#include <util/system/types.h>

#include <format>
#include <memory>
#include <string_view>
#include <vector>


namespace NPerforator::NLinguist::NJvm::NDwarfParser {

// TPath describes a dwarf die.
struct TPath {
    // Substring of translation unit name.
    // It must match exactly one unit.
    std::string Unit;
    // Entity qualified name
    std::vector<std::string> QName;

    std::string ToString() const {
        return std::format("{{Unit = {}, QName = {}}}", Unit, JoinSeq("/", QName));
    }
};

i32 ReadI32Attr(llvm::DWARFDie die, llvm::dwarf::Attribute attr);
ui8 ReadU8Attr(llvm::DWARFDie die, llvm::dwarf::Attribute attr);
size_t ReadSizeTAttr(llvm::DWARFDie die, llvm::dwarf::Attribute attr);

class TParser {
    class TImpl;
public:

    explicit TParser(std::string_view binPath);
    ~TParser();

    // Resolve is not thread-safe
    llvm::DWARFDie Resolve(const TPath& path) const;
private:

    std::unique_ptr<TImpl> Impl_;
};

}
