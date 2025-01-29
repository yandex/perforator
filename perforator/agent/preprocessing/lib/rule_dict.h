#pragma once

#include <perforator/agent/preprocessing/proto/unwind/table.pb.h>

#include <util/generic/algorithm.h>
#include <util/generic/array_ref.h>
#include <util/generic/hash.h>
#include <util/generic/string.h>
#include <util/generic/vector.h>


namespace NPerforator::NBinaryProcessing::NUnwind {

class TRuleDict {
    friend class TRuleDictBuilder;

    explicit TRuleDict(TVector<ui32> mapping, TVector<UnwindRule> rules)
        : Mapping_{std::move(mapping)}
        , Rules_{std::move(rules)}
    {}

public:
    ui32 RemapRule(ui32 id) const {
        return Mapping_.at(id);
    }

    ui32 RuleCount() const {
        return Rules_.size();
    }

    const UnwindRule& GetRule(ui32 id) const {
        return Rules_.at(id);
    }

    TConstArrayRef<UnwindRule> Rules() const {
        return Rules_;
    }

private:
    TVector<ui32> Mapping_;
    TVector<UnwindRule> Rules_;
};

class TRuleDictBuilder {
    struct TRuleInfo {
        ui32 Id = 0;
        ui32 UseCount = 0;
    };

public:
    ui32 Add(UnwindRule&& rule) {
        // NB: Protobuf messages should not be compared this way.
        // But who cares:)
        TString serialized = rule.SerializeAsString();
        if (auto* info = Ids_.FindPtr(serialized)) {
            info->UseCount++;
            return info->Id;
        }

        ui32 id = Rules_.size();
        Rules_.push_back(std::move(rule));
        Ids_[serialized] = TRuleInfo{id, 1};
        return id;
    }

    TRuleDict Finish() && {
        TVector<TRuleInfo> infos(Reserve(Ids_.size()));
        for (auto&& [_, info] : Ids_) {
            infos.emplace_back(std::move(info));
        }

        SortBy(infos, [](auto&& info) {
            return -info.UseCount;
        });

        TVector<ui32> mapping(infos.size());
        TVector<UnwindRule> rules(infos.size());
        for (ui32 i = 0; i < infos.size(); ++i) {
            mapping[infos[i].Id] = i;
            rules[i] = std::move(Rules_[infos[i].Id]);
        }

        return TRuleDict{std::move(mapping), std::move(rules)};
    }

private:
    THashMap<TString, TRuleInfo> Ids_;
    TVector<UnwindRule> Rules_;
};

} // namespace NPerforator::NBinaryProcessing::NUnwind
