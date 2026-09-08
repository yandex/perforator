#include "parser.h"

#include <perforator/lib/llvmex/llvm_exception.h>

#include <contrib/libs/llvm18/include/llvm/DebugInfo/DWARF/DWARFContext.h>
#include <contrib/libs/llvm18/include/llvm/DebugInfo/DWARF/DWARFDie.h>
#include <contrib/libs/llvm18/include/llvm/DebugInfo/DWARF/DWARFFormValue.h>

#include <limits>

#include <util/generic/hash.h>
#include <util/generic/string.h>
#include <util/generic/yexception.h>

namespace NPerforator::NLinguist::NJvm::NDwarfParser {

namespace {

template<typename T>
T ReadAttrImpl(llvm::DWARFDie die, llvm::dwarf::Attribute attr) {
    std::optional<llvm::DWARFFormValue> val = die.find(attr);
    if (!val) {
        std::string name;
        llvm::raw_string_ostream s{name};
        die.getFullName(s);
        ythrow yexception() << "Entity " << name << " does not have expected attribute";
    }

    if (!val->getAsUnsignedConstant().has_value() && !val->getAsSignedConstant().has_value()) {
        ythrow yexception() << "Attribute value is not a constant";
    }
    if (std::is_unsigned_v<T>) {
        auto v = val->getAsUnsignedConstant();
        if (v && *v <= std::numeric_limits<T>::max()) {
            return static_cast<T>(*v);
        }
    } else {
        auto v = val->getAsSignedConstant();
        if (v && *v >= std::numeric_limits<T>::min() && *v <= std::numeric_limits<T>::max()) {
            return static_cast<T>(*v);
        }
    }
    ythrow yexception() << "Attribute value does not fit into desired type";
}

llvm::DWARFDie FindInUnit(llvm::DWARFUnit& unit, const TPath& path) {
    Y_THROW_UNLESS(!path.QName.empty());
    size_t maxMatch = 0;
    llvm::DWARFDie found;
    for (size_t i = 0, cnt = unit.getNumDIEs(); i < cnt; ++i) {
        llvm::DWARFDie die = unit.getDIEAtIndex(i);
        llvm::DWARFDie srcDie = die;

        bool ok = true;
        for (size_t j = path.QName.size() - 1; j < path.QName.size(); --j) {
            const char* name = die.getShortName();
            if (path.QName[j].empty()) {
                if (name != nullptr && !std::string_view(name).empty()) {
                    ok = false;
                    break;
                }
            } else {
                if (name == nullptr || name != path.QName[j]) {
                    ok = false;
                    break;
                }
            }
            maxMatch = std::max(maxMatch, path.QName.size() - j);
            die = die.getParent();
            if (!die.isValid()) {
                ok = false;
                break;
            }
        }
        if (!ok) {
            continue;
        }
        if (die.getTag() != llvm::dwarf::DW_TAG_compile_unit) {
            continue;
        }
        if (found) {
            ythrow yexception() << "Query " << path.ToString() << " matched more than one DIE";
        }
        found = srcDie;
    }
    if (!found) {
        ythrow yexception() << "Query " << path.ToString() << " did not match any DIE (max matched suffix = " << maxMatch << ")";
    }
    return found;
}

}

i32 ReadI32Attr(llvm::DWARFDie die, llvm::dwarf::Attribute attr) {
    return ReadAttrImpl<i32>(die, attr);
}
ui8 ReadU8Attr(llvm::DWARFDie die, llvm::dwarf::Attribute attr) {
    return ReadAttrImpl<ui8>(die, attr);
}
size_t ReadSizeTAttr(llvm::DWARFDie die, llvm::dwarf::Attribute attr) {
    return ReadAttrImpl<size_t>(die, attr);
}

class TParser::TImpl {
public:
    TImpl(
        std::unique_ptr<llvm::DWARFContext> dwarf,
        llvm::object::OwningBinary<llvm::object::ObjectFile> obj
    )
    : Obj_(std::move(obj))
    , Dwarf_(std::move(dwarf)) {
    }

    llvm::DWARFDie Resolve(const TPath& path) const {
        if (!Dwarf_) {
            ythrow yexception() << "Parser is invalid";
        }
        llvm::DWARFUnit& unit = FindUnit(path.Unit);
        return FindInUnit(unit, path);
    }
private:
    llvm::DWARFUnit& FindUnit(std::string_view namePart) const {
        llvm::DWARFUnit* found = nullptr;
        for (const std::unique_ptr<llvm::DWARFUnit>& unit : Dwarf_->getNormalUnitsVector()) {
            llvm::DWARFDie unitDie = unit->getUnitDIE(true);
            const char* uname = unitDie.getShortName();
            Y_THROW_UNLESS(uname != nullptr);
            if (!std::string_view(uname).contains(namePart)) {
                continue;
            }

            if (found != nullptr) {
                ythrow yexception() << "Found more than one unit matching " << namePart;
            }
            found = unit.get();
        }
        if (found == nullptr) {
            ythrow yexception() << "No unit matches " << namePart;
        }
        return *found;
    }

private:
    llvm::object::OwningBinary<llvm::object::ObjectFile> Obj_;
    std::unique_ptr<llvm::DWARFContext> Dwarf_;
};

TParser::TParser(std::string_view binPath) {
    auto elf = Y_LLVM_RAISE(llvm::object::ObjectFile::createObjectFile(binPath));
    auto dwarf = llvm::DWARFContext::create(*elf.getBinary());
    Impl_ = std::make_unique<TImpl>(std::move(dwarf), std::move(elf));
}

llvm::DWARFDie TParser::Resolve(const TPath& path) const {
    return Impl_->Resolve(path);
}

TParser::~TParser() = default;

}
