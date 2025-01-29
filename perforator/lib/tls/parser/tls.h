#pragma once

#include <perforator/lib/tls/magic.h>

#include <util/generic/function_ref.h>
#include <util/generic/strbuf.h>

#include <llvm/Object/ObjectFile.h>


namespace NPerforator::NThreadLocal {

class TTlsParser {
public:
    struct TVariableRef {
        i64 ThreadImageOffset = 0;
        TStringBuf Name;
    };

public:
    explicit TTlsParser(llvm::object::ObjectFile* file);

    void VisitVariables(TFunctionRef<void(const TVariableRef&)> func);

private:
    llvm::object::ObjectFile* File_;
};

} // namespace NPerforator::NThreadLocal
