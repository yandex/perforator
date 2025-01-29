#pragma once

#include "magic.h"

#include <util/generic/maybe.h>
#include <util/generic/string.h>
#include <util/generic/yexception.h>


namespace NPerforator::NThreadLocal {

////////////////////////////////////////////////////////////////////////////////

template <typename T>
struct TTlsRepresentation;

////////////////////////////////////////////////////////////////////////////////

template <typename T>
class TVariable : public TNonCopyable {
    using TRepro = TTlsRepresentation<T>;

public:
    explicit TVariable(T initial = T{}) {
        Set(std::move(initial));
    }

    ~TVariable() {
        Repro_.Clear();
    }

    const T* Get() const& {
        return Value_.Get();
    }

    const T* operator->() const& {
        return Value_.Get();
    }

    void Set(T newValue) {
        Repro_.Clear();
        Value_.ConstructInPlace(std::move(newValue));
        Repro_.Update(Value_.Get());
    }

    void Clear() {
        Repro_.Clear();
        Value_.Clear();
    }

private:
    volatile TMagic Magic_ = MakeMagic(TRepro::Kind);
    TRepro Repro_;
    TMaybe<T> Value_;
};

////////////////////////////////////////////////////////////////////////////////

template <>
struct TTlsRepresentation<ui64> {
public:
    static constexpr EVariableKind Kind = EVariableKind::UnsignedInt64;

    void Update(ui64* newValue) {
        Value = *newValue;
    }

    void Clear() {
        Value = 0;
    }

public:
    std::atomic<ui64> Value = 0;
};

template <>
struct TTlsRepresentation<TStringBuf> {
public:
    static constexpr EVariableKind Kind = EVariableKind::StringPointer;

    void Update(TStringBuf* newValue) {
        Set(newValue->data(), newValue->size());
    }

    void Clear() {
        Set(nullptr, 0);
    }

protected:
    void Set(const char* ptr, size_t size) {
        Size.store(0, std::memory_order_seq_cst);
        Ptr.store(reinterpret_cast<uintptr_t>(ptr), std::memory_order_seq_cst);
        Size.store(size, std::memory_order_seq_cst);
        std::atomic_thread_fence(std::memory_order_release);
    }

public:
    std::atomic<uintptr_t> Ptr = 0;
    std::atomic<size_t> Size = 0;
};

template <>
struct TTlsRepresentation<TString> : public TTlsRepresentation<TStringBuf> {
public:
    void Update(TString* newValue) {
        Set(newValue->data(), newValue->size());
    }
};

////////////////////////////////////////////////////////////////////////////////

#define Y_PERFORATOR_VARIABLE_NAME_PREFIX \
    perforator_tls_

#define Y_PERFORATOR_VARIABLE_NAME_PREFIX_STRING \
    Y_STRINGIZE(Y_PERFORATOR_VARIABLE_NAME_PREFIX)

#define Y_PERFORATOR_THREAD_LOCAL_NAME(name, type) \
    Y_CAT(Y_CAT(Y_PERFORATOR_VARIABLE_NAME_PREFIX, type), Y_CAT(_, name))

#define Y_PERFORATOR_THREAD_LOCAL_GETTER_NAME(name) \
    Y_CAT(PerforatorGetTls, name)

#define Y_PERFORATOR_DEFINE_THREAD_LOCAL(name, type, ...) \
    thread_local __attribute__((used)) \
    ::NPerforator::NThreadLocal::TVariable<type> \
    Y_PERFORATOR_THREAD_LOCAL_NAME(name, type){__VA_ARGS__}; \
\
    ::NPerforator::NThreadLocal::TVariable<type>& \
    Y_PERFORATOR_THREAD_LOCAL_GETTER_NAME(name)() { \
        return Y_PERFORATOR_THREAD_LOCAL_NAME(name, type); \
    } \

#define Y_PERFORATOR_GET_THREAD_LOCAL(name) \
    Y_PERFORATOR_THREAD_LOCAL_GETTER_NAME(name)()

#define Y_PERFORATOR_THREAD_LOCAL_UI64(name, ...) \
    Y_PERFORATOR_DEFINE_THREAD_LOCAL(name, ui64, __VA_ARGS__)

#define Y_PERFORATOR_THREAD_LOCAL_STRING(name, ...) \
    Y_PERFORATOR_DEFINE_THREAD_LOCAL(name, TString, __VA_ARGS__)

#define Y_PERFORATOR_THREAD_LOCAL_STRINGBUF(name, ...) \
    Y_PERFORATOR_DEFINE_THREAD_LOCAL(name, TStringBuf, __VA_ARGS__)

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NThreadLocal
