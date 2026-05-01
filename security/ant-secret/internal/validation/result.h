#pragma once

#include <library/cpp/json/writer/json_value.h>
#include <util/generic/string.h>

namespace NValidation {
    struct TResult {
        bool Valid = false;
        TString System;
        TString Type;
        TString Users;
        TDeque<TString> Owners;
        TString ValidationUrl;
        NJson::TJsonValue AdditionalInfo;

        static TResult FromJson(const NJson::TJsonValue& json) {
            TResult result;
            if (!json["ok"].GetBooleanSafe(false)) {
                // Service can't to deal with it :(
                return result;
            }

            result.Valid = json["valid"].GetBooleanSafe(false);
            if (!result.Valid) {
                // for not valid service doesn't matter what system is it
                return result;
            }

            result.System = json["system"].GetStringSafe("unknown");
            result.Type = json["type"].GetStringSafe("unknown");
            result.Users = json["users"].GetString();
            result.ValidationUrl = json["validation_url"].GetStringSafe(TString());
            result.AdditionalInfo = json["additional_info"];
            if (json.Has("owners")) {
                for (const auto& o: json["owners"].GetArraySafe()) {
                    result.Owners.push_back(o.GetStringSafe());
                }    
            }

            return result;
        }

        static TResult* NewFromJson(const NJson::TJsonValue& json) {
            if (!json["ok"].GetBooleanSafe(false)) {
                // Service can't to deal with it :(
                return nullptr;
            }
            auto result = new TResult;
            result->Valid = json["valid"].GetBooleanSafe(false);
            if (!result->Valid) {
                // for not valid service doesn't matter what system is it
                return result;
            }

            result->System = json["system"].GetStringSafe("unknown");
            result->Type = json["type"].GetStringSafe("unknown");
            result->Users = json["users"].GetString();
            result->ValidationUrl = json["validation_url"].GetStringSafe(TString());
            result->AdditionalInfo = json["additional_info"];
            if (json.Has("owners")) {
                for (const auto& o: json["owners"].GetArraySafe()) {
                    result->Owners.push_back(o.GetStringSafe());
                }
            }

            return result;
        }
    };

}
