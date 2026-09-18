#pragma once

#include <linux/perf_event.h>

#include <cstddef>
#include <memory>

namespace NPerforator::NAgent::NPerfBuffer {

// Round a byte-size request up to the kernel's power-of-two ring pages.
size_t PageCountForBufferSize(size_t bytes, size_t pageSize);

// The record is borrowed and valid only until the callback returns.
// The callback must not throw or reenter the reader. Return false to stop
// consuming the current ring after releasing this record.
using TRecordCallback = bool (*)(void* context, int cpu, const perf_event_header* event);

// Reader for BPF_MAP_TYPE_PERF_EVENT_ARRAY maps. Owns a perf event and
// mapped ring for each CPU online at construction with an index in the map.
// The map fd is borrowed and must outlive the reader. Calls to Poll and
// Consume must be serialized; the reader must be the map's only consumer.
// Only forward-writing events (attr.write_backward == 0) are supported.
class TPerfBuffer {
public:
    TPerfBuffer(
        int mapFd,
        size_t pageCount,
        const perf_event_attr& attr,
        TRecordCallback callback,
        void* context
    );
    ~TPerfBuffer();

    TPerfBuffer(const TPerfBuffer&) = delete;
    TPerfBuffer& operator=(const TPerfBuffer&) = delete;

    // Returns the number of per-CPU rings processed, or a negative errno.
    // Resumes unread rings without blocking, even without a new notification.
    // Each ring is visited at most once per call, including after callback stop.
    int Poll(int timeoutMs);

    // Reads every per-CPU ring through the head observed for that ring.
    // Returns zero on success, or a negative errno.
    int Consume();

private:
    class TImpl;
    std::unique_ptr<TImpl> Impl_;
};

} // namespace NPerforator::NAgent::NPerfBuffer
