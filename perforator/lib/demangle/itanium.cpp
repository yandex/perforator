#include "demangle.h"
#include "itanium.h"

#include <util/generic/scope.h>
#include <util/memory/pool.h>

#include <llvm/Demangle/ItaniumDemangle.h>
#include <llvm/Support/Casting.h>


namespace NPerforator::NDemangle::NPrivate {

using namespace llvm::itanium_demangle;

class TAstNodeAllocator {
public:
    TAstNodeAllocator()
        : Pool_{0}
    {}

    void reset() {
        Pool_.Clear();
    }

    template <typename T, typename ...Args>
    T* makeNode(Args&& ...args) {
        return Pool_.New<T>(std::forward<Args>(args)...);
    }

    void* allocateNodeArray(size_t count) {
        return Pool_.AllocateArray<Node*>(count);
    }

private:
    TMemoryPool Pool_;
};

std::optional<std::string> TryItaniumDemangle(std::string_view str, DemangleOptions options) {
    ManglingParser<TAstNodeAllocator> parser{str.data(), str.data() + str.size()};

    const Node* ast = parser.parse();
    if (ast == nullptr) {
        return std::nullopt;
    }

    if (options.DropVendorSpecificSuffix && ast->getKind() == Node::KDotSuffix) {
        auto* suffix = static_cast<const DotSuffix*>(ast);
        suffix->match([&ast](const Node* prefix, std::string_view) {
            ast = prefix;
        });
    }

    OutputBuffer ob;
    ast->print(ob);
    Y_DEFER {
        std::free(ob.getBuffer());
    };

    // getBufferEnd() points to the last element, not past-the-end.
    return std::string{ob.getBuffer(), ob.getBufferEnd() + 1};
}

} // namespace NPerforator::NDemangle::NPrivate
