# SQUASHFUSE_TOOLS

В этой директории лежит содержимое внешнего ресурса `SQUASHFUSE_TOOLS` для
Linux x86-64 и его декларация для сборочной системы Arcadia.

## Состав

```text
bin/
  squashfuse -> ../squashfuse/bin/squashfuse
  mksquashfs -> ../squashfs-tools/bin/mksquashfs
  unsquashfs -> ../squashfs-tools/bin/unsquashfs
  sqfstar -> ../squashfs-tools/bin/mksquashfs
  sqfscat -> ../squashfs-tools/bin/unsquashfs

squashfuse/
  bin/
    squashfuse
  Documentation/
    CONFIGURATION
    LICENSE
    NEWS
    PLATFORMS
    README
    squashfuse.1
    squashfuse_ll.1

squashfs-tools/
  bin/
    mksquashfs
    unsquashfs
    sqfstar -> mksquashfs
    sqfscat -> unsquashfs
  Documentation/
    4.7.5/
    manpages/
```

`sqfstar` и `sqfscat` — относительные симлинки на основные бинарники.
Это штатные дополнительные режимы `mksquashfs` и `unsquashfs`. Симлинки не
хранятся в Arcadia: `deploy.sh` создаёт их во временном staging-каталоге.

Корневой каталог `bin` генерируется `deploy.sh` только во временном
staging-каталоге и предоставляет единый интерфейс ко всем командам ресурса. Его
можно целиком добавить в `PATH`, не зная внутреннюю раскладку:

```bash
export PATH="$resource_root/bin:$PATH"
```

Все три ELF-бинарника собраны для Linux x86-64 и статически слинкованы.
`squashfuse` strip-нут, а `mksquashfs` и `unsquashfs` оставлены с символами,
как предыдущая сборка squashfs-tools 4.5.1. Они собраны на Ubuntu с glibc,
не с musl.

Контрольные суммы бинарников, фактически попавших в ресурс:

```text
640a8796314415d02e8e79ac41fc02aa2d9a90caf22b614d925d2119d50eb135  squashfuse/bin/squashfuse
e5674e4022526fbbbbebc129aae7e4b76090ac09597b2001af656679ef5c411f  squashfs-tools/bin/mksquashfs
72ad23e366ab13bf3743095b8c614d9d25c8ffb59a63f7cab909788734db87e4  squashfs-tools/bin/unsquashfs
```

## Как собран squashfuse

В ресурс вошёл `squashfuse 0.6.2`, собранный из официального репозитория
`vasi/squashfuse` с поддержкой zlib и Zstd.

Основные зависимости сборки:

```bash
sudo apt-get install -y \
  build-essential \
  autoconf \
  automake \
  libtool \
  pkg-config \
  libfuse-dev \
  zlib1g-dev \
  libzstd-dev
```

Подготовка исходников:

```bash
git clone https://github.com/vasi/squashfuse.git
cd squashfuse
git checkout 0.6.2
./autogen.sh
```

Конфигурация и сборка статических библиотек:

```bash
PKG_CONFIG="pkg-config --static" \
CFLAGS="-O2" \
./configure \
  --disable-shared \
  --enable-static \
  --disable-demo \
  --disable-multithreading

make -j4
```

Libtool убирал `-static` из финальной команды линковки, поэтому исполняемый
файл перелинковывался напрямую:

```bash
gcc -static -O2 \
  -o squashfuse-static-zstd \
  squashfuse-hl.o \
  .libs/libsquashfuse_convenience.a \
  .libs/libfuseprivate.a \
  -lz \
  -lzstd \
  -lfuse \
  -pthread

strip squashfuse-static-zstd
```

Проверка результата:

```bash
file squashfuse-static-zstd
ldd squashfuse-static-zstd || true
readelf -l squashfuse-static-zstd | grep INTERP || true
readelf -d squashfuse-static-zstd | grep NEEDED || true
sha256sum squashfuse-static-zstd
```

Бинарник прошёл реальный mount/unmount Zstd SquashFS-образа на Linux через
`/dev/fuse`. Для работы на воркере всё равно нужны доступный `/dev/fuse`,
права на FUSE-mount и `fusermount` либо `fusermount3`.

## Как собраны squashfs-tools

`mksquashfs` и `unsquashfs` собраны из официального релиза
`plougher/squashfs-tools 4.7.5` на Ubuntu 24.04 x86-64. Включены gzip, LZO,
LZ4, XZ и Zstd. Как и в предыдущей сборке 4.5.1, компрессором по умолчанию
выбран gzip.

Зависимости:

```bash
sudo apt-get install -y \
  build-essential \
  pkg-config \
  zlib1g-dev \
  liblzo2-dev \
  liblz4-dev \
  liblzma-dev \
  libzstd-dev \
  curl \
  ca-certificates
```

Получение исходников:

```bash
curl -fL \
  https://github.com/plougher/squashfs-tools/archive/refs/tags/4.7.5.tar.gz \
  -o squashfs-tools-4.7.5.tar.gz

tar -xzf squashfs-tools-4.7.5.tar.gz
cd squashfs-tools-4.7.5/squashfs-tools
```

Сборка:

```bash
make clean
make -j"$(nproc)" \
  EXTRA_LDFLAGS=-static \
  GZIP_SUPPORT=1 \
  LZO_SUPPORT=1 \
  LZ4_SUPPORT=1 \
  XZ_SUPPORT=1 \
  ZSTD_SUPPORT=1 \
  COMP_DEFAULT=gzip

ln -s mksquashfs sqfstar
ln -s unsquashfs sqfscat
```

Проверялись тип ELF, отсутствие динамических зависимостей и реальный цикл
создания и распаковки Zstd SquashFS:

```bash
file mksquashfs unsquashfs
ldd mksquashfs || true
ldd unsquashfs || true

./mksquashfs source-dir test.squashfs \
  -comp zstd \
  -Xcompression-level 10 \
  -noappend \
  -repro-time 0

./unsquashfs -d extracted test.squashfs
cmp source-dir/path/to/file extracted/path/to/file
```

Для проверки были созданы и распакованы образы с каждым из поддерживаемых
компрессоров: `gzip`, `lzo`, `lz4`, `xz` и `zstd`.

## Как был создан архив

Для подготовки симлинков и архива используется:

```bash
./deploy.sh
```

Скрипт создаёт корневой `bin` и остальные симлинки только во временном
staging-каталоге, создаёт `squashfs-tools.tgz` и загружает его в Sandbox.
Дополнительные аргументы передаются в `ya upload`; для проверки без публикации
можно выполнить `./deploy.sh --dry-run`. Путь к архиву переопределяется через
`SQUASHFS_TOOLS_ARTIFACT`.

SHA-256 опубликованного архива:

```text
3a84c02cecb627f75b91de6e69691656581355044547c30049d06322682020db  squashfs-tools.tgz
```

## Как ресурс опубликован

Команда публикации:

```bash
./deploy.sh
```

Текущий ресурс:

- Sandbox resource ID: `12918939924`;
- URI для `resources.json`: `sbr:12918939924`;
- тип: `NOTS_EXTERNAL_RESOURCE`;
- владелец: `FRONTEND_BUILD_PLATFORM`;
- TTL: `INF`;
- platform attribute: `linux_x64`;
- страница: `https://sandbox.yandex-team.ru/resource/12918939924/view`;
- download URL: `https://proxy.sandbox.yandex-team.ru/12918939924`;
- upload task: `https://sandbox.yandex-team.ru/task/4417693276/view`.

Атрибуты ресурса:

```text
platform=linux_x64
squashfuse=0.6.2
mksquashfs=4.7.5
unsquashfs=4.7.5
sqfstar=4.7.5
sqfscat=4.7.5
```

В `resources.json` ресурс сопоставлен платформе сборочной системы
`linux-x86_64`, а в `ya.make` объявлен bundle `SQUASHFUSE_TOOLS`.
