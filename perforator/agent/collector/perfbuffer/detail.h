#pragma once

#include <linux/perf_event.h>
#include <sys/epoll.h>

#include <cstddef>
#include <cstdint>
#include <functional>
#include <string_view>
#include <vector>

namespace NPerforator::NAgent::NPerfBuffer::NDetail {

std::vector<int> ParseCpuList(std::string_view input);

// Register before publishing the fd to a producer that may immediately write.
void RegisterAndPublishPerfEvent(
    int epollFd,
    int perfFd,
    uint32_t index,
    const std::function<void()>& publish
);

using TRecordCallback = bool (*)(void* context, int cpu, const perf_event_header* event);

int ConsumeRing(
    void* mapping,
    size_t pageSize,
    size_t ringSize,
    int cpu,
    TRecordCallback callback,
    void* context,
    std::vector<std::byte>& scratch
);

struct TPerfRing {
    void* Mapping;
    size_t PageSize;
    size_t RingSize;
    int Cpu;
    std::vector<std::byte> Scratch;
    bool Pending = false;
};

// epoll_wait semantics, except errors are returned as negative errno values.
using TWaitCallback = int (*)(void* context, epoll_event* events, int maxEvents, int timeoutMs);

// Reads borrowed mappings. Keeping notification handling separate from resource
// ownership also allows testing Poll without perf/BPF privileges.
class TReader {
public:
    TReader(
        std::vector<TPerfRing> rings,
        TRecordCallback callback,
        void* context,
        TWaitCallback wait,
        void* waitContext
    );

    int Poll(int timeoutMs);
    int Consume();

private:
    void Enqueue(size_t index);
    int ConsumePending();

private:
    std::vector<TPerfRing> Rings_;
    TRecordCallback Callback_;
    void* Context_;
    TWaitCallback Wait_;
    void* WaitContext_;
    std::vector<epoll_event> ReadyEvents_;
    std::vector<size_t> Pending_;
};

} // namespace NPerforator::NAgent::NPerfBuffer::NDetail
