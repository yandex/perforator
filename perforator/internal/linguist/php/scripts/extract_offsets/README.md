# PHP Offset Extraction

Extracts struct field offsets from PHP (Zend Engine) source code for use in the eBPF unwinder.

## Usage

```bash
# Build
ya make -r perforator/internal/linguist/php/scripts/extract_offsets

# Run (pass path to offsets.c in source tree)
./extract_offsets --php-version 7.4.0 --offsets-c perforator/internal/linguist/php/scripts/extract_offsets/offsets.c

# Save to JSON
./extract_offsets --php-version 7.4.0 --offsets-c path/to/offsets.c > path/to/php-7.4.0-offsets.json

# Generate all PHP 7.x
for v in 7.0.0 7.1.0 7.2.0 7.3.0 7.4.0; do
    ./extract_offsets --php-version "$v" \
        --offsets-c perforator/internal/linguist/php/scripts/extract_offsets/offsets.c \
        > "perforator/internal/linguist/php/agent/offsets/php-${v}-offsets.json"
done
```

## How it works

Uses the shared `common/scripts/extract_offsets_lib.py` which:
1. Downloads PHP source from `github.com/php/php-src` (cached in `~/.offset_sources/`)
2. Runs `./buildconf` + `./configure --disable-all` to generate headers
3. Compiles `offsets.c` against PHP headers using `offsetof()`
4. Parses output and formats as JSON
