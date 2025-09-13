#pragma once

#include <util/system/types.h>
#include <util/generic/array_ref.h>

#include <memory>

namespace NPerforator::NProto::NPProf {
class Profile;
}

namespace NPerforator::NStacksSampling {

class TAggregatingSampler final {
public:
    TAggregatingSampler(ui64 rate);
    ~TAggregatingSampler();

    void AddProfile(TArrayRef<const char> profileBytes);

    void AddProfile(const NPerforator::NProto::NPProf::Profile& profile);

    const NPerforator::NProto::NPProf::Profile& GetResultingProfile() const noexcept;

private:
    ui64 Rate_;
    ui64 SeenStacks_{0};

    class Impl;
    std::unique_ptr<Impl> Impl_;
};

}
