#pragma once

#include "attrs.h"
#include "sections.h"


#define BPF_EMPTY_ARG

#define BPF_MAP_DEF_UINT(name, val) int (*name)[val]
#define BPF_MAP_DEF_TYPE(name, val) typeof(val) *name
#define BPF_MAP_DEF_ARRAY(name, val) typeof(val) *name[]

#define BPF_MAP_STRUCT(NAME, TYPE, KEY, VALUE, SIZE, FLAGS) \
    struct NAME { \
        BPF_MAP_DEF_UINT(type, TYPE); \
        BPF_MAP_DEF_TYPE(key, KEY); \
        BPF_MAP_DEF_TYPE(value, VALUE); \
        BPF_MAP_DEF_UINT(max_entries, SIZE); \
        BPF_MAP_DEF_UINT(map_flags, FLAGS); \
    }

#define BPF_MAP_ANON_STRUCT(TYPE, KEY, VALUE, SIZE, FLAGS) \
    BPF_MAP_STRUCT(BPF_EMPTY_ARG, TYPE, KEY, VALUE, SIZE, FLAGS)

#define BPF_MAP(NAME, TYPE, KEY, VALUE, SIZE) \
    BPF_MAP_ANON_STRUCT(TYPE, KEY, VALUE, SIZE, 0) NAME SEC(BPF_SEC_BTF_MAPS);

#define BPF_MAP_F(NAME, TYPE, KEY, VALUE, SIZE, FLAGS) \
    BPF_MAP_ANON_STRUCT(TYPE, KEY, VALUE, SIZE, FLAGS) NAME SEC(BPF_SEC_BTF_MAPS);
