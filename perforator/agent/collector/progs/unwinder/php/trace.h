#define PHP_TRACE(fmt, ...) \
    BPF_TRACE("php: " fmt "\n", ##__VA_ARGS__)
