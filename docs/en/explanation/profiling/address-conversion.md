# Address entities and conversions between them

One of the crucial things we need to understand in order to collect correct profiles is the relationship between different address representations used in profiling. This document explains the key concepts and conversion formulas between ELF offset, ELF vaddr, and VMA addr.

## Key Terms and Definitions

### ELF Offset
**ELF offset** (also called file offset) is the position of data within the ELF file on disk. It represents the byte offset from the beginning of the file where specific code or data is located. This is a static value that doesn't change when the program is loaded into memory.

### ELF Virtual Address (ELF vaddr)
**ELF vaddr** is the virtual address as defined in the ELF file's program headers. It represents the intended virtual address where a segment should be loaded in memory. This address is specified in the ELF file and can be found in the `p_vaddr` field of program headers.

### Virtual Memory Area Address (VMA address)
**VMA addr** is the actual virtual address where the code is loaded in the process memory space. This is the runtime address that appears in profiling samples and can differ from ELF vaddr due to:
- Address Space Layout Randomization (ASLR)
- Position Independent Executables (PIE)
- Dynamic library loading

## Address Conversion Formulas

Understanding these conversions is essential for accurate profiling, especially when dealing with multiple executable mappings or Position Independent Executables.

### ELF offset ↔ ELF vaddr

To perform these conversions, you first need to find the appropriate program header segment that contains the address or offset you're working with.

**Finding the correct program header:**
- For ELF offset: find program header where `p_offset ≤ ELF_offset < p_offset + p_filesz`
- For ELF vaddr: find program header where `p_vaddr ≤ ELF_vaddr < p_vaddr + p_memsz`

**ELF offset → ELF vaddr:**
```
ELF_vaddr = ELF_offset - p_offset + p_vaddr
```

**ELF vaddr → ELF offset:**
```
ELF_offset = ELF_vaddr - p_vaddr + p_offset
```

Where:
- `p_offset` is the file offset of the program header segment
- `p_vaddr` is the virtual address of the program header segment
- `p_filesz` is the size of the segment in the file
- `p_memsz` is the size of the segment in memory

### VMA addr ↔ ELF vaddr

**VMA addr → ELF vaddr:**
```
ELF_vaddr = VMA_addr - base_address
```

**ELF addr → VMA addr:**
```
VMA_addr = ELF_vaddr + base_address
```

Where:
- `base_address` is the runtime base address of the mapping

**Finding the base address:**
According to the [ELF specification](https://refspecs.linuxbase.org/elf/gabi4+/ch5.pheader.html), the base address is calculated from three values: the virtual memory load address, the maximum page size, and the lowest virtual address of a program's loadable segment. To compute the base address:

1. Find the memory address associated with the lowest `p_vaddr` value for a `PT_LOAD` segment
2. Round this address down to the nearest multiple of the maximum page size
3. Round the corresponding `p_vaddr` value down to the nearest multiple of the maximum page size
4. The base address is the difference between the truncated memory address and the truncated `p_vaddr` value

For most practical purposes in profiling, this can be calculated using one of two approaches:

**General approach:**
```
base_address = lowest_vma_address - lowest_loadable_segment_vaddr
```

**Another approach:**

```
base_address = any_vma_address - corresponding_elf_vaddr
```

We can also find the base address using this formula because the base address represents a constant difference between VMA addresses and ELF address space. In Perforator, we use this second formula as it's more convenient for our use case.

**Important note about alignment:**
When calculating `lowest_loadable_segment_vaddr` and `lowest_loadable_executable_segment_vaddr`, you must account for alignment. According to the ELF specification, `p_vaddr` should be aligned with `p_align`, and the alignment requirement is:
```
p_vaddr % p_align == p_offset % p_align
```

The actual aligned virtual address used in calculations should be:
```
aligned_vaddr = p_vaddr & ~(p_align - 1)
```

### VMA addr ↔ ELF offset

**VMA addr → ELF offset:**
```
ELF_offset = VMA_addr - base_address - p_vaddr + p_offset
```

**ELF offset → VMA addr:**
```
VMA_addr = ELF_offset - p_offset + p_vaddr + base_address
```

## Multiple Executable Mappings

Executables can be loaded into process memory via multiple executable mappings (e.g., BOLT-optimized binaries). In such cases:

1. **Track all executable loadable program headers** instead of just the first one
2. **Calculate base address using the lowest executable VMA** across all mappings
3. **Validate alignment invariants** for each program header:
   ```
   p_vaddr % p_align == p_offset % p_align
   ```

## Practical Examples

### Example 1: Position Independent Executable (PIE)

Consider the binary (PIE) with the following executable segment:

```
Program Header (executable segment):
  Type:     LOAD
  Offset:   0x0000000000bd4fd4
  VirtAddr: 0x0000000000bd5fd4
  FileSiz:  0x0000000002f41d78
  MemSiz:   0x0000000002f41d78
  Flags:    R E
  Align:    0x1000

Runtime mapping:
  VMA start: 0x55a1c0bd5000
  VMA end:   0x55a1c3f17000
```

Calculations:
```
# Apply alignment
aligned_vaddr = 0x0000000000bd5fd4 & ~(0x1000 - 1) = 0x0000000000bd5000

# Calculate base address
base_address = 0x55a1c0bd5000 - 0x0000000000bd5000 = 0x55a1c0000000

# Convert addresses for instruction at VMA 0x55a1c0bd6500:
ELF_vaddr = 0x55a1c0bd6500 - 0x55a1c0000000 = 0x0000000000bd6500
ELF_offset = 0x0000000000bd6500 - 0x0000000000bd5fd4 + 0x0000000000bd4fd4 = 0x0000000000bd5000
```

### Example 2: Static Binary

Consider a static binary loaded at its intended address:

```
Program Header (executable segment):
  Type:     LOAD
  Offset:   0x1000
  VirtAddr: 0x401000
  FileSiz:  0x2000
  MemSiz:   0x2000
  Flags:    R E
  Align:    0x1000

Runtime mapping:
  VMA start: 0x401000  # Same as ELF vaddr
  VMA end:   0x403000
```

Calculations:
```
# Apply alignment
aligned_vaddr = 0x401000 & ~(0x1000 - 1) = 0x401000

# Calculate base address
base_address = 0x401000 - 0x401000 = 0x0 (no offset)

# Convert addresses for instruction at VMA 0x401500:
ELF_vaddr = 0x401500 - 0x0 = 0x401500 (same as VMA)
ELF_offset = 0x401500 - 0x401000 + 0x1000 = 0x1500
```

## Usage in Profiling

During profiling:
1. **Sample collection** uses VMA addr (runtime addresses)
2. **Symbolization** requires ELF offset (file addresses)
3. **Conversion** happens using the formulas above with mapping information

This ensures accurate mapping between runtime samples and source code locations.
