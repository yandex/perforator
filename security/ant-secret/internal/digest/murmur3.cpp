#include "murmur3.h"

namespace NDigest {
    namespace {
        #define Murmur3_ROT32(x, y) ((x << y) | (x >> (32 - y)))

        uint32_t read32(const uint8_t* p) {
            uint32_t res;
            std::memcpy(&res, p, sizeof(res));
            return res;
        }

        uint32_t fmix32(uint32_t h) {
            h ^= h >> 16;
            h *= 0x85ebca6b;
            h ^= h >> 13;
            h *= 0xc2b2ae35;
            h ^= h >> 16;
            return h;
        }
    }

    uint32_t Murmur3_32(TStringBuf in, uint32_t seed) {
        return Murmur3_32(in.data(), in.size(), seed);
    }

    uint32_t Murmur3_32(const void* key, size_t len, uint32_t seed) {
        const uint8_t* data = static_cast<const uint8_t*>(key);

        const uint32_t c1 = 0xcc9e2d51;
        const uint32_t c2 = 0x1b873593;

        uint32_t h = seed;
        size_t i = 0;
        for (; i + 8 <= len; i += 8) {
            uint32_t k1 = read32(data + i);
            uint32_t k2 = read32(data + i + 4);

            k1 *= c1;
            k1 = Murmur3_ROT32(k1, 15);
            k1 *= c2;
            h ^= k1;
            h = Murmur3_ROT32(h, 13);
            h = h * 5 + 0xe6546b64;

            k2 *= c1;
            k2 = Murmur3_ROT32(k2, 15);
            k2 *= c2;
            h ^= k2;
            h = Murmur3_ROT32(h, 13);
            h = h * 5 + 0xe6546b64;
        }

        for (; i + 4 <= len; i += 4) {
            uint32_t k1 = read32(data + i);

            k1 *= c1;
            k1 = Murmur3_ROT32(k1, 15);
            k1 *= c2;

            h ^= k1;
            h = Murmur3_ROT32(h, 13);
            h = h * 5 + 0xe6546b64;
        }

        uint32_t k1 = 0;
        const uint8_t* tail = data + i;
        switch (len & 3) {
            case 3: k1 ^= uint32_t(tail[2]) << 16;
            case 2: k1 ^= uint32_t(tail[1]) << 8;
            case 1:
                k1 ^= uint32_t(tail[0]);
                k1 *= c1;
                k1 = Murmur3_ROT32(k1, 15);
                k1 *= c2;
                h ^= k1;
        }

        h ^= static_cast<uint32_t>(len);
        return fmix32(h);
    }
}
