#include "ehframe.h"

#include <perforator/agent/preprocessing/lib/analyze.h>

#include <util/stream/buffer.h>


extern "C" struct raw_binary_analysis build_binary_analysis(const char* path) try {
    auto analysis = NPerforator::NBinaryProcessing::AnalyzeBinary(path);

    TBufferOutput out;

    NPerforator::NBinaryProcessing::SerializeBinaryAnalysis(std::move(analysis), &out);

    auto&& buf = out.Buffer();
    char* res = new char[buf.size()];
    memcpy(res, buf.data(), buf.size());
    return {res, nullptr, static_cast<int>(buf.size())};
} catch (const std::exception& err) {
    return {nullptr, new TString{err.what()}, 0};
} catch (...) {
    return {nullptr, new TString{"Unknown error"}, 0};
}

extern "C" void binary_analysis_free(char* analysis) {
    delete[] analysis;
}

extern "C" const char* binary_analysis_error_text(void* err) {
    return static_cast<TString*>(err)->data();
}

extern "C" void binary_analysis_free_error(void* err) {
    delete static_cast<TString*>(err);
}
