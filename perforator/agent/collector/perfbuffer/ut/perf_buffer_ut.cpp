#include <perforator/agent/collector/perfbuffer/detail.h>
#include <perforator/agent/collector/perfbuffer/perf_buffer.h>

#include <library/cpp/testing/unittest/registar.h>

#include <util/generic/yexception.h>
#include <util/system/file.h>

#include <linux/perf_event.h>
#include <sys/eventfd.h>

#include <algorithm>
#include <cerrno>
#include <cstddef>
#include <cstring>
#include <limits>
#include <utility>
#include <vector>

namespace NPerforator::NAgent::NPerfBuffer {
namespace {

struct TRecord {
    perf_event_header Header;
    ui64 Payload;
};

struct TCallbackState {
    TVector<int> Cpus;
    TVector<TRecord> Records;
};

bool CaptureRecord(void* context, int cpu, const perf_event_header* event) {
    auto& state = *static_cast<TCallbackState*>(context);
    UNIT_ASSERT_VALUES_EQUAL(event->size, sizeof(TRecord));
    state.Cpus.push_back(cpu);
    TRecord record{};
    std::memcpy(&record, event, sizeof(record));
    state.Records.push_back(record);
    return true;
}

bool CaptureOneRecord(void* context, int cpu, const perf_event_header* event) {
    CaptureRecord(context, cpu, event);
    return false;
}

void WriteWrapped(
    std::byte* ring,
    size_t ringSize,
    size_t offset,
    const void* source,
    size_t size)
{
    const size_t first = std::min(size, ringSize - offset);
    std::memcpy(ring + offset, source, first);
    std::memcpy(ring, static_cast<const std::byte*>(source) + first, size - first);
}

struct TRing {
    static constexpr size_t PageSize = 4096;
    static constexpr size_t RingSize = 4096;
    std::vector<std::byte> Mapping = std::vector<std::byte>(PageSize + RingSize);
    std::vector<std::byte> Scratch;

    perf_event_mmap_page* Metadata() {
        return reinterpret_cast<perf_event_mmap_page*>(Mapping.data());
    }

    std::byte* Data() {
        return Mapping.data() + PageSize;
    }

    void Append(ui64 payload) {
        const TRecord record{{PERF_RECORD_SAMPLE, 0, sizeof(TRecord)}, payload};
        WriteWrapped(Data(), RingSize, Metadata()->data_head & (RingSize - 1), &record, sizeof(record));
        Metadata()->data_head += sizeof(record);
    }

    int Consume(TRecordCallback callback, void* context) {
        return NDetail::ConsumeRing(Mapping.data(), PageSize, RingSize, 7, callback, context, Scratch);
    }
};

struct TWaitState {
    std::vector<ui32> Ready;
    std::vector<int> Timeouts;
    int Error = 0;

    static int Wait(void* context, epoll_event* events, int maxEvents, int timeoutMs) {
        auto& state = *static_cast<TWaitState*>(context);
        state.Timeouts.push_back(timeoutMs);
        if (state.Error) {
            return std::exchange(state.Error, 0);
        }
        UNIT_ASSERT(state.Ready.size() <= static_cast<size_t>(maxEvents));
        const int count = static_cast<int>(state.Ready.size());
        for (int i = 0; i < count; ++i) {
            events[i] = {};
            events[i].events = EPOLLIN;
            events[i].data.u32 = state.Ready[i];
        }
        // Like perf_poll, consume the notification independently of ring data.
        state.Ready.clear();
        return count;
    }
};

std::vector<NDetail::TPerfRing> DescribeRings(std::initializer_list<TRing*> rings) {
    std::vector<NDetail::TPerfRing> result;
    for (auto* ring : rings) {
        result.push_back({ring->Mapping.data(), TRing::PageSize, TRing::RingSize, static_cast<int>(result.size()), {}});
    }
    return result;
}

NDetail::TReader MakeReader(std::initializer_list<TRing*> rings, TCallbackState& state, TWaitState& wait) {
    return NDetail::TReader(DescribeRings(rings), CaptureOneRecord, &state, TWaitState::Wait, &wait);
}

} // anonymous namespace

Y_UNIT_TEST_SUITE(TCpuListTest) {
    Y_UNIT_TEST(ParseRangesAndDuplicates) {
        const auto cpus = NDetail::ParseCpuList(" 0-2, 4,2,7-8\n");
        UNIT_ASSERT_VALUES_EQUAL(cpus.size(), 6);
        UNIT_ASSERT_VALUES_EQUAL(cpus[0], 0);
        UNIT_ASSERT_VALUES_EQUAL(cpus[1], 1);
        UNIT_ASSERT_VALUES_EQUAL(cpus[2], 2);
        UNIT_ASSERT_VALUES_EQUAL(cpus[3], 4);
        UNIT_ASSERT_VALUES_EQUAL(cpus[4], 7);
        UNIT_ASSERT_VALUES_EQUAL(cpus[5], 8);
    }

    Y_UNIT_TEST(RejectsMalformedLists) {
        UNIT_ASSERT_EXCEPTION(NDetail::ParseCpuList("3-1"), yexception);
        UNIT_ASSERT_EXCEPTION(NDetail::ParseCpuList("0,"), yexception);
        UNIT_ASSERT_EXCEPTION(NDetail::ParseCpuList("cpu0"), yexception);
    }
} // Y_UNIT_TEST_SUITE(TCpuListTest)

Y_UNIT_TEST_SUITE(TPerfBufferInitializationTest) {
    Y_UNIT_TEST(RejectsBackwardWritesBeforeInspectingMap) {
        perf_event_attr attr{};
        attr.write_backward = 1;
        // No valid BPF map or privileges are needed: validation must run first.
        UNIT_ASSERT_EXCEPTION_CONTAINS(
            TPerfBuffer(0, 1, attr, CaptureRecord, nullptr),
            yexception, "backward perf buffer writes are not supported");
    }

    Y_UNIT_TEST(RegistersBeforePublishingAndReadsInitialRecord) {
        const TFileHandle epollFd{epoll_create1(EPOLL_CLOEXEC)};
        const TFileHandle notificationFd{eventfd(0, EFD_CLOEXEC | EFD_NONBLOCK)};
        UNIT_ASSERT(epollFd.IsOpen());
        UNIT_ASSERT(notificationFd.IsOpen());
        TRing ring;
        TCallbackState state;

        NDetail::RegisterAndPublishPerfEvent(epollFd, notificationFd, 0, [&] {
            // eventfd does not clear readiness in poll like perf does. Check
            // registration explicitly at publication, before producing data.
            epoll_event event{};
            event.events = EPOLLIN;
            UNIT_ASSERT_VALUES_EQUAL(epoll_ctl(epollFd, EPOLL_CTL_ADD, notificationFd, &event), -1);
            UNIT_ASSERT_VALUES_EQUAL(errno, EEXIST);
            ring.Append(42);
            UNIT_ASSERT_VALUES_EQUAL(eventfd_write(notificationFd, 1), 0);
        });

        struct TWaitContext {
            int EpollFd;
            int NotificationFd;

            static int Wait(void* context, epoll_event* events, int maxEvents, int timeoutMs) {
                const auto& self = *static_cast<TWaitContext*>(context);
                const int ready = epoll_wait(self.EpollFd, events, maxEvents, timeoutMs);
                if (ready < 0) {
                    return -errno;
                }
                if (ready > 0) {
                    eventfd_t count = 0;
                    UNIT_ASSERT_VALUES_EQUAL(eventfd_read(self.NotificationFd, &count), 0);
                    UNIT_ASSERT_VALUES_EQUAL(count, 1);
                }
                return ready;
            }
        } wait{epollFd, notificationFd};
        NDetail::TReader reader(DescribeRings({&ring}), CaptureRecord, &state, TWaitContext::Wait, &wait);

        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(100), 1);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 1);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[0].Payload, 42);
        UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, ring.Metadata()->data_head);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(0), 0);
    }

    Y_UNIT_TEST(DoesNotPublishIfRegistrationFails) {
        const TFileHandle epollFd{epoll_create1(EPOLL_CLOEXEC)};
        UNIT_ASSERT(epollFd.IsOpen());
        bool published = false;
        UNIT_ASSERT_EXCEPTION_CONTAINS(
            NDetail::RegisterAndPublishPerfEvent(epollFd, -1, 0, [&] { published = true; }),
            yexception, "failed to add per-CPU perf event to epoll");
        UNIT_ASSERT(!published);
    }
} // Y_UNIT_TEST_SUITE(TPerfBufferInitializationTest)

Y_UNIT_TEST_SUITE(TPerfRingTest) {
    Y_UNIT_TEST(RoundsRequestedSizeLikeTheGoReader) {
        UNIT_ASSERT_VALUES_EQUAL(PageCountForBufferSize(1, 4096), 1);
        UNIT_ASSERT_VALUES_EQUAL(PageCountForBufferSize(4096, 4096), 1);
        UNIT_ASSERT_VALUES_EQUAL(PageCountForBufferSize(4097, 4096), 2);
        UNIT_ASSERT_VALUES_EQUAL(PageCountForBufferSize(12 * 1024 * 1024, 4096), 4096);
        UNIT_ASSERT_VALUES_EQUAL(PageCountForBufferSize(16 * 1024 * 1024, 4096), 4096);
        UNIT_ASSERT_VALUES_EQUAL(PageCountForBufferSize(65537, 65536), 2);
        UNIT_ASSERT_EXCEPTION(PageCountForBufferSize(0, 4096), yexception);
        UNIT_ASSERT_EXCEPTION(PageCountForBufferSize(4096, 0), yexception);
        UNIT_ASSERT_EXCEPTION(PageCountForBufferSize(std::numeric_limits<size_t>::max(), 4096), yexception);
    }

    Y_UNIT_TEST(RejectsMappingLengthOverflowIncludingMetadataPage) {
        const size_t highestPowerOfTwo = size_t{1} << (std::numeric_limits<size_t>::digits - 1);
        UNIT_ASSERT_VALUES_EQUAL(PageCountForBufferSize(highestPowerOfTwo, 4096), highestPowerOfTwo / 4096);
        UNIT_ASSERT_EXCEPTION(PageCountForBufferSize(highestPowerOfTwo + 1, 4096), yexception);
        UNIT_ASSERT_EXCEPTION(PageCountForBufferSize(1, std::numeric_limits<size_t>::max()), yexception);

        perf_event_attr attr{};
        UNIT_ASSERT_EXCEPTION_CONTAINS(
            TPerfBuffer(0, highestPowerOfTwo, attr, CaptureRecord, nullptr),
            yexception, "perf buffer mapping is too large");
    }

    Y_UNIT_TEST(ReleasesProcessedRecordsBeforeNextCallback) {
        constexpr size_t pageSize = 4096;
        constexpr size_t ringSize = 4096;
        std::vector<std::byte> mapping(pageSize + ringSize);
        auto* metadata = reinterpret_cast<perf_event_mmap_page*>(mapping.data());
        const TRecord records[] = {
            {{PERF_RECORD_SAMPLE, 0, sizeof(TRecord)}, 11},
            {{PERF_RECORD_SAMPLE, 0, sizeof(TRecord)}, 22},
        };
        std::memcpy(mapping.data() + pageSize, records, sizeof(records));
        metadata->data_head = sizeof(records);
        struct TProgress {
            perf_event_mmap_page* Metadata;
            size_t Processed = 0;
            bool Released = true;
        } progress{metadata};
        auto callback = [](void* context, int, const perf_event_header*) {
            auto& state = *static_cast<TProgress*>(context);
            state.Released &= state.Metadata->data_tail == state.Processed * sizeof(TRecord);
            ++state.Processed;
            return true;
        };
        std::vector<std::byte> scratch;
        UNIT_ASSERT_VALUES_EQUAL(NDetail::ConsumeRing(mapping.data(), pageSize, ringSize, 0, callback, &progress, scratch), 0);
        UNIT_ASSERT_C(progress.Released, "previously processed records still occupy producer ring capacity");
        UNIT_ASSERT_VALUES_EQUAL(progress.Processed, 2);
        UNIT_ASSERT_VALUES_EQUAL(metadata->data_tail, sizeof(records));
    }

    Y_UNIT_TEST(ConsumesContiguousAndWrappedRecords) {
        constexpr size_t pageSize = 4096;
        constexpr size_t ringSize = 4096;
        std::vector<std::byte> mapping(pageSize + ringSize);
        auto* metadata = reinterpret_cast<perf_event_mmap_page*>(mapping.data());
        auto* ring = mapping.data() + pageSize;

        const TRecord first{{PERF_RECORD_SAMPLE, 0, sizeof(TRecord)}, 11};
        const TRecord second{{PERF_RECORD_LOST, 0, sizeof(TRecord)}, 22};
        const uint64_t tail = ringSize - sizeof(TRecord) / 2;
        WriteWrapped(ring, ringSize, tail & (ringSize - 1), &first, sizeof(first));
        WriteWrapped(ring, ringSize, (tail + sizeof(first)) & (ringSize - 1), &second, sizeof(second));
        metadata->data_tail = tail;
        metadata->data_head = tail + sizeof(first) + sizeof(second);

        TCallbackState state;
        std::vector<std::byte> scratch;
        UNIT_ASSERT_VALUES_EQUAL(
            NDetail::ConsumeRing(
                mapping.data(), pageSize, ringSize, 7, CaptureRecord, &state, scratch),
            0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 2);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[0].Payload, 11);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[1].Payload, 22);
        UNIT_ASSERT_VALUES_EQUAL(state.Cpus[0], 7);
        UNIT_ASSERT_VALUES_EQUAL(metadata->data_tail, metadata->data_head);
    }

    Y_UNIT_TEST(RejectsCorruptRecord) {
        constexpr size_t pageSize = 4096;
        constexpr size_t ringSize = 4096;
        std::vector<std::byte> mapping(pageSize + ringSize);
        auto* metadata = reinterpret_cast<perf_event_mmap_page*>(mapping.data());
        auto* header = reinterpret_cast<perf_event_header*>(mapping.data() + pageSize);
        header->size = 0;
        metadata->data_head = sizeof(*header);

        TCallbackState state;
        std::vector<std::byte> scratch;
        UNIT_ASSERT_VALUES_EQUAL(
            NDetail::ConsumeRing(
                mapping.data(), pageSize, ringSize, 0, CaptureRecord, &state, scratch),
            -EBADMSG);
        UNIT_ASSERT_VALUES_EQUAL(metadata->data_tail, metadata->data_head);
        UNIT_ASSERT(state.Records.empty());
    }

    Y_UNIT_TEST(CallbackCanStopAtARecordBoundary) {
        constexpr size_t pageSize = 4096;
        constexpr size_t ringSize = 4096;
        std::vector<std::byte> mapping(pageSize + ringSize);
        auto* metadata = reinterpret_cast<perf_event_mmap_page*>(mapping.data());
        auto* ring = mapping.data() + pageSize;
        const TRecord records[] = {
            {{PERF_RECORD_SAMPLE, 0, sizeof(TRecord)}, 11},
            {{PERF_RECORD_SAMPLE, 0, sizeof(TRecord)}, 22},
        };
        std::memcpy(ring, records, sizeof(records));
        metadata->data_head = sizeof(records);

        TCallbackState state;
        std::vector<std::byte> scratch;
        UNIT_ASSERT_VALUES_EQUAL(
            NDetail::ConsumeRing(
                mapping.data(), pageSize, ringSize, 0, CaptureOneRecord, &state, scratch),
            0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 1);
        UNIT_ASSERT_VALUES_EQUAL(metadata->data_tail, sizeof(TRecord));
        UNIT_ASSERT_VALUES_EQUAL(
            NDetail::ConsumeRing(
                mapping.data(), pageSize, ringSize, 0, CaptureOneRecord, &state, scratch),
            0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 2);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[1].Payload, 22);
        UNIT_ASSERT_VALUES_EQUAL(metadata->data_tail, metadata->data_head);
    }

    Y_UNIT_TEST(BorrowsContiguousRecordsUntilCallbackReturns) {
        TRing ring;
        ring.Append(11);
        ring.Append(22);
        struct TState {
            TRing* Ring;
            size_t Processed = 0;
        } state{&ring};
        auto callback = [](void* context, int, const perf_event_header* event) {
            auto& state = *static_cast<TState*>(context);
            const size_t offset = state.Processed * sizeof(TRecord);
            UNIT_ASSERT_VALUES_EQUAL(state.Ring->Metadata()->data_tail, offset);
            UNIT_ASSERT(event == reinterpret_cast<const perf_event_header*>(state.Ring->Data() + offset));
            UNIT_ASSERT(state.Ring->Scratch.empty());
            TRecord record{};
            std::memcpy(&record, event, sizeof(record));
            UNIT_ASSERT_VALUES_EQUAL(record.Payload, state.Processed == 0 ? 11 : 22);
            ++state.Processed;
            return true;
        };
        UNIT_ASSERT_VALUES_EQUAL(ring.Consume(callback, &state), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Processed, 2);
        UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, ring.Metadata()->data_head);
    }

    Y_UNIT_TEST(CopiesWrappedHeadersAndPayloadsBeforeCallback) {
        // Exercise both a split header and a split payload.
        for (size_t offset : {TRing::RingSize - sizeof(perf_event_header) / 2,
                              TRing::RingSize - sizeof(perf_event_header)}) {
            TRing ring;
            ring.Metadata()->data_tail = offset;
            ring.Metadata()->data_head = offset;
            ring.Append(42);
            auto callback = [](void* context, int cpu, const perf_event_header* event) {
                auto& ring = *static_cast<TRing*>(context);
                UNIT_ASSERT_VALUES_EQUAL(cpu, 7);
                UNIT_ASSERT(event == reinterpret_cast<const perf_event_header*>(ring.Scratch.data()));
                UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, ring.Metadata()->data_head - sizeof(TRecord));
                TRecord record{};
                std::memcpy(&record, event, sizeof(record));
                UNIT_ASSERT_VALUES_EQUAL(record.Payload, 42);
                return false;
            };
            UNIT_ASSERT_VALUES_EQUAL(ring.Consume(callback, &ring), 0);
            UNIT_ASSERT_VALUES_EQUAL(ring.Scratch.size(), sizeof(TRecord));
            UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, ring.Metadata()->data_head);
        }
    }

    Y_UNIT_TEST(ConsumesOnlyTheObservedHead) {
        TRing ring;
        ring.Append(11);
        auto callback = [](void* context, int, const perf_event_header*) {
            static_cast<TRing*>(context)->Append(22);
            return true;
        };
        UNIT_ASSERT_VALUES_EQUAL(ring.Consume(callback, &ring), 0);
        UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, sizeof(TRecord));
        UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_head, 2 * sizeof(TRecord));
        TCallbackState state;
        UNIT_ASSERT_VALUES_EQUAL(ring.Consume(CaptureRecord, &state), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 1);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[0].Payload, 22);
    }

    Y_UNIT_TEST(HandlesEmptyRingAndWrappingCounters) {
        TRing ring;
        TCallbackState state;
        UNIT_ASSERT_VALUES_EQUAL(ring.Consume(CaptureRecord, &state), 0);
        UNIT_ASSERT(state.Records.empty());
        ring.Metadata()->data_tail = std::numeric_limits<ui64>::max() - sizeof(perf_event_header) + 1;
        ring.Metadata()->data_head = ring.Metadata()->data_tail;
        ring.Append(42);
        UNIT_ASSERT_VALUES_EQUAL(ring.Consume(CaptureRecord, &state), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 1);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[0].Payload, 42);
        UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, ring.Metadata()->data_head);
    }

    Y_UNIT_TEST(RejectsTruncatedAndOversizedRecords) {
        for (size_t size : {size_t{0}, sizeof(perf_event_header) - 1,
                            sizeof(TRecord), TRing::RingSize + 1}) {
            TRing ring;
            const perf_event_header header{PERF_RECORD_SAMPLE, 0, static_cast<ui16>(size)};
            std::memcpy(ring.Data(), &header, sizeof(header));
            ring.Metadata()->data_head = sizeof(header);
            TCallbackState state;
            UNIT_ASSERT_VALUES_EQUAL(ring.Consume(CaptureRecord, &state), -EBADMSG);
            UNIT_ASSERT(state.Records.empty());
            UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, ring.Metadata()->data_head);
        }
    }

    Y_UNIT_TEST(RejectsTruncatedHeaderAfterValidRecord) {
        TRing ring;
        ring.Append(11);
        ring.Metadata()->data_head += sizeof(perf_event_header) - 1;
        TCallbackState state;
        UNIT_ASSERT_VALUES_EQUAL(ring.Consume(CaptureRecord, &state), -EBADMSG);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 1);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[0].Payload, 11);
        UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, ring.Metadata()->data_head);
    }

    Y_UNIT_TEST(RecoversAfterOverrun) {
        TRing ring;
        ring.Metadata()->data_head = TRing::RingSize + sizeof(TRecord);
        TCallbackState state;
        UNIT_ASSERT_VALUES_EQUAL(ring.Consume(CaptureRecord, &state), -EOVERFLOW);
        UNIT_ASSERT(state.Records.empty());
        UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, ring.Metadata()->data_head);
        ring.Append(42);
        UNIT_ASSERT_VALUES_EQUAL(ring.Consume(CaptureRecord, &state), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 1);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[0].Payload, 42);
    }

    Y_UNIT_TEST(ConsumesCompletelyFullRing) {
        TRing ring;
        constexpr size_t recordCount = TRing::RingSize / sizeof(TRecord);
        for (size_t i = 0; i < recordCount; ++i) {
            ring.Append(i);
        }
        TCallbackState state;
        UNIT_ASSERT_VALUES_EQUAL(ring.Consume(CaptureRecord, &state), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), recordCount);
        for (size_t i = 0; i < recordCount; ++i) {
            UNIT_ASSERT_VALUES_EQUAL(state.Records[i].Payload, i);
        }
        UNIT_ASSERT(ring.Scratch.empty());
        UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, TRing::RingSize);
    }

    Y_UNIT_TEST(RejectsInvalidRingConfiguration) {
        TRing ring;
        TCallbackState state;
        for (size_t size : {size_t{0}, sizeof(perf_event_header) - 1, TRing::RingSize - 1}) {
            UNIT_ASSERT_VALUES_EQUAL(
                NDetail::ConsumeRing(ring.Mapping.data(), TRing::PageSize, size, 0, CaptureRecord, &state, ring.Scratch),
                -EINVAL);
        }
        UNIT_ASSERT_VALUES_EQUAL(
            NDetail::ConsumeRing(nullptr, TRing::PageSize, TRing::RingSize, 0, CaptureRecord, &state, ring.Scratch),
            -EINVAL);
        UNIT_ASSERT_VALUES_EQUAL(
            NDetail::ConsumeRing(ring.Mapping.data(), 0, TRing::RingSize, 0, CaptureRecord, &state, ring.Scratch),
            -EINVAL);
        UNIT_ASSERT_VALUES_EQUAL(
            NDetail::ConsumeRing(ring.Mapping.data(), TRing::PageSize, TRing::RingSize, 0, nullptr, &state, ring.Scratch),
            -EINVAL);
        UNIT_ASSERT(state.Records.empty());
    }
} // Y_UNIT_TEST_SUITE(TPerfRingTest)

Y_UNIT_TEST_SUITE(TPerfPollTest) {
    Y_UNIT_TEST(ResumesStoppedRingWithoutAnotherNotification) {
        TRing ring;
        ring.Append(11);
        ring.Append(22);
        ring.Append(33);
        TCallbackState state;
        TWaitState wait;
        wait.Ready = {0};
        auto reader = MakeReader({&ring}, state, wait);

        const int timeouts[] = {42, -1, 5000};
        for (size_t i = 0; i < 3; ++i) {
            UNIT_ASSERT_VALUES_EQUAL(reader.Poll(timeouts[i]), 1);
            UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), i + 1);
            UNIT_ASSERT_VALUES_EQUAL(state.Records.back().Payload, 11 * (i + 1));
            UNIT_ASSERT_VALUES_EQUAL(ring.Metadata()->data_tail, sizeof(TRecord) * (i + 1));
            UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), i == 0 ? 42 : 0);
            UNIT_ASSERT(wait.Ready.empty());
        }
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(42), 0);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 42);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 3);
    }

    Y_UNIT_TEST(PendingRingDoesNotStarveAnotherCpuOrRunTwicePerCall) {
        TRing first;
        first.Append(11);
        first.Append(22);
        first.Append(33);
        TRing second;
        second.Append(44);
        TCallbackState state;
        TWaitState wait;
        wait.Ready = {0};
        auto reader = MakeReader({&first, &second}, state, wait);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(42), 1);

        // A new notification for the pending ring must not enqueue it twice.
        wait.Ready = {0, 1};
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(-1), 2);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 3);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[1].Payload, 22);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[2].Payload, 44);
        UNIT_ASSERT_VALUES_EQUAL(state.Cpus[1], 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Cpus[2], 1);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(-1), 1);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.back().Payload, 33);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(42), 0);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 42);
    }

    Y_UNIT_TEST(ResumesAfterExplicitConsumeWithoutNotification) {
        TRing ring;
        ring.Append(11);
        ring.Append(22);
        TCallbackState state;
        TWaitState wait;
        auto reader = MakeReader({&ring}, state, wait);
        UNIT_ASSERT_VALUES_EQUAL(reader.Consume(), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 1);
        UNIT_ASSERT(wait.Timeouts.empty());
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(-1), 1);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 2);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.back().Payload, 22);
    }

    Y_UNIT_TEST(ExplicitConsumeClearsPendingRing) {
        TRing ring;
        ring.Append(11);
        ring.Append(22);
        TCallbackState state;
        TWaitState wait;
        wait.Ready = {0};
        auto reader = MakeReader({&ring}, state, wait);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(42), 1);
        UNIT_ASSERT_VALUES_EQUAL(reader.Consume(), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 2);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(42), 0);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 42);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 2);
    }

    Y_UNIT_TEST(PreservesRemainingNotificationsAfterRingError) {
        TRing corrupt;
        corrupt.Metadata()->data_head = sizeof(perf_event_header);
        TRing valid;
        valid.Append(11);
        TCallbackState state;
        TWaitState wait;
        wait.Ready = {0, 1};
        auto reader = MakeReader({&corrupt, &valid}, state, wait);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(42), -EBADMSG);
        UNIT_ASSERT(state.Records.empty());
        UNIT_ASSERT(wait.Ready.empty());
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(-1), 1);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 1);
        UNIT_ASSERT_VALUES_EQUAL(state.Records[0].Payload, 11);
        UNIT_ASSERT_VALUES_EQUAL(state.Cpus[0], 1);
    }

    Y_UNIT_TEST(PreservesPendingRingAfterInterruptedWait) {
        TRing ring;
        ring.Append(11);
        ring.Append(22);
        TCallbackState state;
        TWaitState wait;
        wait.Ready = {0};
        auto reader = MakeReader({&ring}, state, wait);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(42), 1);
        wait.Error = -EINTR;
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(-1), -EINTR);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 1);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(-1), 1);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.size(), 2);
        UNIT_ASSERT_VALUES_EQUAL(state.Records.back().Payload, 22);
    }

    Y_UNIT_TEST(KeepsRecordsPublishedAfterHeadSnapshot) {
        TRing ring;
        ring.Append(11);
        struct TState {
            TRing* Ring;
            TCallbackState Captured;
        } state{&ring, {}};
        auto callback = [](void* context, int cpu, const perf_event_header* event) {
            auto& state = *static_cast<TState*>(context);
            CaptureRecord(&state.Captured, cpu, event);
            if (state.Captured.Records.size() == 1) {
                state.Ring->Append(22);
            }
            return true;
        };
        TWaitState wait;
        wait.Ready = {0};
        NDetail::TReader reader(DescribeRings({&ring}), callback, &state, TWaitState::Wait, &wait);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(42), 1);
        UNIT_ASSERT_VALUES_EQUAL(state.Captured.Records.size(), 1);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(-1), 1);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 0);
        UNIT_ASSERT_VALUES_EQUAL(state.Captured.Records.size(), 2);
        UNIT_ASSERT_VALUES_EQUAL(state.Captured.Records.back().Payload, 22);
        UNIT_ASSERT_VALUES_EQUAL(reader.Poll(42), 0);
        UNIT_ASSERT_VALUES_EQUAL(wait.Timeouts.back(), 42);
    }
} // Y_UNIT_TEST_SUITE(TPerfPollTest)

} // namespace NPerforator::NAgent::NPerfBuffer
