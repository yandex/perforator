#include "analyzer_impl.h"

namespace NPerforator::NLinguist::NJvm {

TJvmAnalysis ProcessOffsetRegistry(const TJvmMetadata& metadata, TOffsetRegistryAnalysisOptions options, ui32 version) {
    TJvmAnalysis offsets;
    NPerforator::NBinaryProcessing::NJvm::Cheatsheet& s = offsets.Cheatsheet;
    s.set_code_blob_name(metadata.FindFieldOffset("CodeBlob", "_name"));
    s.set_code_blob_frame_size(
        metadata.FindFieldOffset("CodeBlob", "_frame_size")
    );

    s.set_code_heap_log2_segment_size(
        metadata.FindFieldOffset("CodeHeap", "_log2_segment_size")
    );
    s.set_code_heap_memory(metadata.FindFieldOffset("CodeHeap", "_memory"));
    s.set_code_heap_segmap(metadata.FindFieldOffset("CodeHeap", "_segmap"));

    s.set_heap_block_header_length(
        metadata.FindFieldOffset("HeapBlock::Header", "_length")
    );
    s.set_heap_block_header_used(
        metadata.FindFieldOffset("HeapBlock::Header", "_used")
    );
    s.set_heap_block_header(metadata.FindFieldOffset("HeapBlock", "_header"));
    s.set_heap_block_allocated_space(metadata.FindTypeSize("HeapBlock"));

    s.set_constant_pool_pool_holder(
        metadata.FindFieldOffset("ConstantPool", "_pool_holder")
    );
    s.set_constant_pool_base_offset(metadata.FindTypeSize("ConstantPool"));

    s.set_const_method_constants(
        metadata.FindFieldOffset("ConstMethod", "_constants")
    );
    s.set_const_method_name_index(
        metadata.FindFieldOffset("ConstMethod", "_name_index")
    );

    s.set_method_const_method(
        metadata.FindFieldOffset("Method", "_constMethod")
    );

    if (version >= 22) {
        s.set_nmethod_method(metadata.FindFieldOffset("nmethod", "_method"));
    } else {
        s.set_nmethod_method(metadata.FindFieldOffset("CompiledMethod", "_method"));
    }

    s.set_growable_array_data(
        metadata.FindFieldOffset("GrowableArray<int>", "_data")
    );
    s.set_growable_array_length(
        metadata.FindFieldOffset("GrowableArrayBase", "_len")
    );

    s.set_klass_name(metadata.FindFieldOffset("Klass", "_name"));

    s.set_stub_queue_stub_buffer(
        metadata.FindFieldOffset("StubQueue", "_stub_buffer")
    );
    s.set_stub_queue_buffer_limit(
        metadata.FindFieldOffset("StubQueue", "_buffer_limit")
    );

    s.set_symbol_body(metadata.FindFieldOffset("Symbol", "_body[0]"));
    s.set_symbol_length(metadata.FindFieldOffset("Symbol", "_length"));

    s.set_virtual_space_low(metadata.FindFieldOffset("VirtualSpace", "_low"));
    s.set_virtual_space_high(
        metadata.FindFieldOffset("VirtualSpace", "_high")
    );

    if (options.IncludeAddresses) {
        s.set_code_cache_heaps(
            metadata.FindStaticFieldAddress("CodeCache", "_heaps")
        );
        s.set_abstract_interpreter_code(
            metadata.FindStaticFieldAddress("AbstractInterpreter", "_code")
        );
    }

    return offsets;
}

}
