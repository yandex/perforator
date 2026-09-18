#include "perf_buffer.h"

#include "detail.h"

#include <util/generic/yexception.h>

#include <linux/bpf.h>

#include <sys/epoll.h>
#include <sys/mman.h>
#include <sys/syscall.h>
#include <unistd.h>

#include <algorithm>
#include <cerrno>
#include <charconv>
#include <climits>
#include <cstdint>
#include <cstring>
#include <fcntl.h>
#include <memory>
#include <limits>
#include <string>
#include <system_error>
#include <utility>
#include <vector>

namespace NPerforator::NAgent::NPerfBuffer {

size_t PageCountForBufferSize(size_t bytes, size_t pageSize) {
    Y_ENSURE(bytes != 0 && pageSize != 0, "perf buffer and page sizes must be non-zero");
    const size_t requested = bytes / pageSize + (bytes % pageSize != 0);
    const size_t maxPages = std::numeric_limits<size_t>::max() / pageSize - 1;
    size_t pages = 1;
    while (pages < requested) {
        Y_ENSURE(pages <= maxPages / 2, "perf buffer size overflows mmap length");
        pages *= 2;
    }
    Y_ENSURE(pages <= maxPages, "perf buffer size overflows mmap length");
    return pages;
}

namespace {

constexpr std::string_view OnlineCpusPath = "/sys/devices/system/cpu/online";

[[noreturn]] void ThrowError(std::string_view operation, int error) {
    ythrow yexception()
        << operation << ": errno " << error << " (" << std::strerror(error) << ')';
}

[[noreturn]] void ThrowLastError(std::string_view operation) {
    ThrowError(operation, errno);
}

uint64_t PointerValue(const void* pointer) {
    return static_cast<uint64_t>(reinterpret_cast<uintptr_t>(pointer));
}

int Bpf(enum bpf_cmd command, union bpf_attr& attr) {
    return static_cast<int>(syscall(SYS_bpf, command, &attr, sizeof(attr)));
}

bpf_map_info GetMapInfo(int mapFd) {
    bpf_map_info info{};
    union bpf_attr attr {};
    attr.info.bpf_fd = mapFd;
    attr.info.info_len = sizeof(info);
    attr.info.info = PointerValue(&info);
    if (Bpf(BPF_OBJ_GET_INFO_BY_FD, attr) < 0) {
        ThrowLastError("failed to inspect perf event array map");
    }
    if (info.type != BPF_MAP_TYPE_PERF_EVENT_ARRAY) {
        ythrow yexception()
            << "expected BPF_MAP_TYPE_PERF_EVENT_ARRAY, got map type " << info.type;
    }
    return info;
}

void UpdateMap(int mapFd, int key, int value) {
    union bpf_attr attr {};
    attr.map_fd = mapFd;
    attr.key = PointerValue(&key);
    attr.value = PointerValue(&value);
    attr.flags = BPF_ANY;
    if (Bpf(BPF_MAP_UPDATE_ELEM, attr) < 0) {
        ThrowLastError("failed to install perf event fd into BPF map");
    }
}

void DeleteMapEntry(int mapFd, int key) noexcept {
    union bpf_attr attr {};
    attr.map_fd = mapFd;
    attr.key = PointerValue(&key);
    Bpf(BPF_MAP_DELETE_ELEM, attr);
}

std::string ReadFile(std::string_view path) {
    const std::string nullTerminatedPath{path};
    const int fd = open(nullTerminatedPath.c_str(), O_RDONLY | O_CLOEXEC);
    if (fd < 0) {
        ThrowLastError("failed to open CPU list");
    }

    std::string result;
    char buffer[256];
    for (;;) {
        const ssize_t count = read(fd, buffer, sizeof(buffer));
        if (count > 0) {
            result.append(buffer, static_cast<size_t>(count));
            continue;
        }
        const int error = errno;
        if (count == 0) {
            close(fd);
            return result;
        }
        if (error == EINTR) {
            continue;
        }
        close(fd);
        ThrowError("failed to read CPU list", error);
    }
}

int OpenPerfEvent(const perf_event_attr& sourceAttr, int cpu) {
    perf_event_attr attr = sourceAttr;
    const int fd = static_cast<int>(syscall(
        SYS_perf_event_open,
        &attr,
        -1,
        cpu,
        -1,
        PERF_FLAG_FD_CLOEXEC));
    if (fd < 0) {
        ThrowLastError("failed to open per-CPU perf event");
    }
    return fd;
}

} // anonymous namespace

namespace NDetail {

void RegisterAndPublishPerfEvent(
    int epollFd,
    int perfFd,
    uint32_t index,
    const std::function<void()>& publish
)
{
    epoll_event event{};
    event.events = EPOLLIN;
    event.data.u32 = index;
    if (epoll_ctl(epollFd, EPOLL_CTL_ADD, perfFd, &event) < 0) {
        ThrowLastError("failed to add per-CPU perf event to epoll");
    }
    // EPOLL_CTL_ADD calls perf_poll, which clears any existing notification.
    // The producer must not see the fd until registration has completed.
    publish();
}

std::vector<int> ParseCpuList(std::string_view input) {
    std::vector<int> result;
    size_t position = 0;

    auto skipWhitespace = [&] {
        while (position < input.size() &&
               (input[position] == ' ' || input[position] == '\t' ||
                input[position] == '\r' || input[position] == '\n')) {
            ++position;
        }
    };
    auto parseCpu = [&] {
        skipWhitespace();
        unsigned value = 0;
        const char* begin = input.data() + position;
        const char* end = input.data() + input.size();
        const auto parsed = std::from_chars(begin, end, value);
        if (parsed.ec != std::errc{} || parsed.ptr == begin || value > INT_MAX) {
            ythrow yexception() << "invalid CPU list near offset " << position;
        }
        position = static_cast<size_t>(parsed.ptr - input.data());
        return static_cast<int>(value);
    };

    skipWhitespace();
    while (position < input.size()) {
        const int first = parseCpu();
        skipWhitespace();
        int last = first;
        if (position < input.size() && input[position] == '-') {
            ++position;
            last = parseCpu();
            if (last < first) {
                ythrow yexception() << "descending CPU range " << first << '-' << last;
            }
        }
        for (int cpu = first;; ++cpu) {
            result.push_back(cpu);
            if (cpu == last) {
                break;
            }
        }
        skipWhitespace();
        if (position == input.size()) {
            break;
        }
        if (input[position] != ',') {
            ythrow yexception() << "invalid CPU list separator near offset " << position;
        }
        ++position;
        skipWhitespace();
        if (position == input.size()) {
            ythrow yexception() << "trailing comma in CPU list";
        }
    }

    std::sort(result.begin(), result.end());
    result.erase(std::unique(result.begin(), result.end()), result.end());
    return result;
}

int ConsumeRing(
    void* mapping,
    size_t pageSize,
    size_t ringSize,
    int cpu,
    TRecordCallback callback,
    void* context,
    std::vector<std::byte>& scratch)
{
    if (!mapping || !callback || pageSize == 0 || ringSize == 0 ||
        (ringSize & (ringSize - 1)) != 0) {
        return -EINVAL;
    }

    auto* metadata = static_cast<perf_event_mmap_page*>(mapping);
    auto* ring = reinterpret_cast<std::byte*>(mapping) + pageSize;
    const uint64_t head = __atomic_load_n(&metadata->data_head, __ATOMIC_ACQUIRE);
    uint64_t tail = metadata->data_tail;
    const size_t mask = ringSize - 1;

    if (head - tail > ringSize) {
        __atomic_store_n(&metadata->data_tail, head, __ATOMIC_RELEASE);
        return -EOVERFLOW;
    }

    while (tail != head) {
        const size_t offset = static_cast<size_t>(tail & mask);
        perf_event_header header{};
        if (offset + sizeof(header) <= ringSize) {
            std::memcpy(&header, ring + offset, sizeof(header));
        } else {
            const size_t first = ringSize - offset;
            std::memcpy(&header, ring + offset, first);
            std::memcpy(reinterpret_cast<std::byte*>(&header) + first,
                        ring,
                        sizeof(header) - first);
        }

        if (header.size < sizeof(header) || header.size > ringSize || header.size > head - tail) {
            __atomic_store_n(&metadata->data_tail, head, __ATOMIC_RELEASE);
            return -EBADMSG;
        }

        const perf_event_header* event = nullptr;
        if (offset + header.size <= ringSize) {
            event = reinterpret_cast<const perf_event_header*>(ring + offset);
        } else {
            scratch.resize(header.size);
            const size_t first = ringSize - offset;
            std::memcpy(scratch.data(), ring + offset, first);
            std::memcpy(scratch.data() + first, ring, header.size - first);
            event = reinterpret_cast<const perf_event_header*>(scratch.data());
        }

        tail += header.size;
        const bool keepReading = callback(context, cpu, event);
        // The callback may borrow ring memory. Release this record as
        // soon as it returns, before processing the rest of the batch.
        __atomic_store_n(&metadata->data_tail, tail, __ATOMIC_RELEASE);
        if (!keepReading) {
            break;
        }
    }

    __atomic_store_n(&metadata->data_tail, tail, __ATOMIC_RELEASE);
    return 0;
}

TReader::TReader(
    std::vector<TPerfRing> rings,
    TRecordCallback callback,
    void* context,
    TWaitCallback wait,
    void* waitContext)
    : Rings_{std::move(rings)}
    , Callback_{callback}
    , Context_{context}
    , Wait_{wait}
    , WaitContext_{waitContext}
    , ReadyEvents_(Rings_.size())
{
    // A pass may requeue each ring once before removing the processed prefix.
    Pending_.reserve(2 * Rings_.size());
}

void TReader::Enqueue(size_t index) {
    if (!Rings_[index].Pending) {
        Pending_.push_back(index);
        Rings_[index].Pending = true;
    }
}

int TReader::Poll(int timeoutMs) {
    // perf_poll clears the kernel notification even if records remain unread.
    // Still collect new notifications so pending rings cannot starve other CPUs.
    const int ready = Wait_(
        WaitContext_, ReadyEvents_.data(), static_cast<int>(ReadyEvents_.size()),
        Pending_.empty() ? timeoutMs : 0);
    if (ready < 0) {
        return ready;
    }
    for (int i = 0; i < ready; ++i) {
        const size_t index = ReadyEvents_[i].data.u32;
        if (index >= Rings_.size()) {
            return -EBADMSG;
        }
        Enqueue(index);
    }
    return ConsumePending();
}

int TReader::ConsumePending() {
    const size_t count = Pending_.size();
    size_t processed = 0;
    int error = 0;
    // Only process the initial batch. A callback returning false must not be
    // called again for the same ring until the next Poll/Consume invocation.
    while (processed < count) {
        const size_t index = Pending_[processed++];
        auto& ring = Rings_[index];
        ring.Pending = false;
        error = ConsumeRing(
            ring.Mapping, ring.PageSize, ring.RingSize, ring.Cpu,
            Callback_, Context_, ring.Scratch);
        if (error < 0) {
            break;
        }
        const auto* metadata = static_cast<perf_event_mmap_page*>(ring.Mapping);
        if (__atomic_load_n(&metadata->data_head, __ATOMIC_ACQUIRE) != metadata->data_tail) {
            Enqueue(index);
        }
    }
    // Preserve both requeued rings and the unprocessed part of a failed batch.
    Pending_.erase(Pending_.begin(), Pending_.begin() + processed);
    return error < 0 ? error : static_cast<int>(count);
}

int TReader::Consume() {
    for (size_t index = 0; index < Rings_.size(); ++index) {
        Enqueue(index);
    }
    const int result = ConsumePending();
    return result < 0 ? result : 0;
}

} // namespace NDetail

class TPerfBuffer::TImpl {
private:
    struct TCpuBuffer {
        int Cpu = -1;
        int MapKey = -1;
        int Fd = -1;
        void* Mapping = MAP_FAILED;
        bool InstalledInMap = false;
    };

public:
    ~TImpl() {
        Reader_.reset();
        for (const auto& buffer : Buffers_) {
            if (buffer->InstalledInMap) {
                DeleteMapEntry(MapFd_, buffer->MapKey);
            }
            if (buffer->Fd >= 0) {
                syscall(SYS_ioctl, buffer->Fd, PERF_EVENT_IOC_DISABLE, 0);
            }
            if (buffer->Mapping != MAP_FAILED) {
                munmap(buffer->Mapping, PageSize_ + RingSize_);
            }
            if (buffer->Fd >= 0) {
                close(buffer->Fd);
            }
        }
        if (EpollFd_ >= 0) {
            close(EpollFd_);
        }
    }

    void Initialize(
        int mapFd,
        size_t pageCount,
        const perf_event_attr& attr,
        TRecordCallback callback,
        void* context)
    {
        if (mapFd < 0 || pageCount == 0 || (pageCount & (pageCount - 1)) != 0 || !callback) {
            ThrowError("invalid perf buffer configuration", EINVAL);
        }
        if (attr.write_backward) {
            ThrowError("backward perf buffer writes are not supported", EINVAL);
        }

        MapFd_ = mapFd;
        const long pageSize = sysconf(_SC_PAGESIZE);
        if (pageSize <= 0) {
            ThrowLastError("failed to resolve system page size");
        }
        PageSize_ = static_cast<size_t>(pageSize);
        if (pageCount > (SIZE_MAX / PageSize_) - 1) {
            ThrowError("perf buffer mapping is too large", EOVERFLOW);
        }
        RingSize_ = PageSize_ * pageCount;

        const bpf_map_info mapInfo = GetMapInfo(MapFd_);
        const std::vector<int> onlineCpus = NDetail::ParseCpuList(ReadFile(OnlineCpusPath));

        EpollFd_ = epoll_create1(EPOLL_CLOEXEC);
        if (EpollFd_ < 0) {
            ThrowLastError("failed to create perf buffer epoll instance");
        }

        for (const int cpu : onlineCpus) {
            if (static_cast<uint32_t>(cpu) >= mapInfo.max_entries) {
                continue;
            }
            auto buffer = std::make_unique<TCpuBuffer>();
            buffer->Cpu = cpu;
            buffer->MapKey = cpu;
            Buffers_.push_back(std::move(buffer));
            TCpuBuffer& current = *Buffers_.back();

            current.Fd = OpenPerfEvent(attr, cpu);
            current.Mapping = mmap(
                nullptr,
                PageSize_ + RingSize_,
                PROT_READ | PROT_WRITE,
                MAP_SHARED,
                current.Fd,
                0);
            if (current.Mapping == MAP_FAILED) {
                ThrowLastError("failed to mmap per-CPU perf event ring");
            }
            if (syscall(SYS_ioctl, current.Fd, PERF_EVENT_IOC_ENABLE, 0) < 0) {
                ThrowLastError("failed to enable per-CPU perf event");
            }

            NDetail::RegisterAndPublishPerfEvent(
                EpollFd_, current.Fd, static_cast<uint32_t>(Buffers_.size() - 1), [&] {
                    UpdateMap(MapFd_, current.MapKey, current.Fd);
                    current.InstalledInMap = true;
                });
        }

        if (Buffers_.empty()) {
            ThrowError("perf event array has no entries for online CPUs", ENODEV);
        }
        std::vector<NDetail::TPerfRing> rings;
        rings.reserve(Buffers_.size());
        for (const auto& buffer : Buffers_) {
            rings.push_back({buffer->Mapping, PageSize_, RingSize_, buffer->Cpu, {}});
        }
        Reader_ = std::make_unique<NDetail::TReader>(std::move(rings), callback, context, Wait, this);
    }

    int Poll(int timeoutMs) {
        return Reader_->Poll(timeoutMs);
    }

    int Consume() {
        return Reader_->Consume();
    }

private:
    static int Wait(void* context, epoll_event* events, int maxEvents, int timeoutMs) {
        const auto* self = static_cast<TImpl*>(context);
        const int ready = epoll_wait(self->EpollFd_, events, maxEvents, timeoutMs);
        return ready < 0 ? -errno : ready;
    }

private:
    int MapFd_ = -1;
    int EpollFd_ = -1;
    size_t PageSize_ = 0;
    size_t RingSize_ = 0;
    std::vector<std::unique_ptr<TCpuBuffer>> Buffers_;
    std::unique_ptr<NDetail::TReader> Reader_;
};

TPerfBuffer::TPerfBuffer(
    int mapFd,
    size_t pageCount,
    const perf_event_attr& attr,
    TRecordCallback callback,
    void* context)
    : Impl_{std::make_unique<TImpl>()}
{
    Impl_->Initialize(mapFd, pageCount, attr, callback, context);
}

TPerfBuffer::~TPerfBuffer() = default;

int TPerfBuffer::Poll(int timeoutMs) {
    return Impl_->Poll(timeoutMs);
}

int TPerfBuffer::Consume() {
    return Impl_->Consume();
}

} // namespace NPerforator::NAgent::NPerfBuffer
