# SQUASHFS_TOOLS

В директории объявлен внешний Linux x86-64 resource с `mksquashfs` и
`unsquashfs` 4.7.5 для hermetic `node_modules.layer`.

## Состав ресурса

```text
bin/
  squashfuse -> ../squashfuse/bin/squashfuse
  mksquashfs -> ../squashfs-tools/bin/mksquashfs
  unsquashfs -> ../squashfs-tools/bin/unsquashfs
  sqfstar -> ../squashfs-tools/bin/mksquashfs
  sqfscat -> ../squashfs-tools/bin/unsquashfs
squashfuse/bin/squashfuse
squashfs-tools/bin/{mksquashfs,unsquashfs,sqfstar,sqfscat}
squashfs-tools/lib/libzstd.so.1
mksquashfs -> squashfs-tools/bin/mksquashfs
unsquashfs -> squashfs-tools/bin/unsquashfs
sqfstar -> squashfs-tools/bin/mksquashfs
sqfscat -> squashfs-tools/bin/unsquashfs
```

`squashfuse` и совместимые entrypoint-симлинки сохранены для mount-flow и
ручных сценариев. Симлинки создаются только во временном staging-каталоге и не
попадают в коммиты Arcadia.

`mksquashfs` и `unsquashfs` динамически связаны с glibc и `libzstd.so.1`. RUNPATH
`$ORIGIN/../lib` заставляет их использовать Zstd из ресурса. Бинарники собраны
через Ubuntu 20 SDK; максимальная требуемая версия символов glibc — 2.14.

Полностью статическая glibc здесь недопустима: при разрешении владельца файлов
она динамически загружает NSS-модули хоста. На машинах с `cauth_userd` это
приводило к segfault (DEVTOOLSSUPPORT-94602). Динамическая сборка успешно
работает с `cauth_userd`, не меняя владельцев файлов в layer.

Включён только Zstd, компрессор по умолчанию — Zstd. Исходники — официальный
релиз `plougher/squashfs-tools 4.7.5`; `libzstd.so.1` собрана из vendored
`contrib/libs/zstd` и `contrib/libs/xxhash`. Сборка и линковка выполнены с
sysroot `build/platform/linux_sdk/ubuntu_20`.

Контрольные суммы проверенной сборки:

```text
258791ea9c30625f992043c2c698e405b66fabc0b61b4204bb6d2cee8d785aef  bin/mksquashfs
3528fee4dab0a6247f9de685cbeec92b64bc10a6bd59be44bf62b15231e39565  bin/unsquashfs
12e4801e0b1db8e4771ad840786f87ce2ff0d1fbec0a3dd224d6f5b078c8d32a  lib/libzstd.so.1
```

Сборка проверена на Linux-машине с `passwd/group: files cauth_userd`: создание
и распаковка Zstd-образа успешно работают без `-all-root` и `-q`.

## Публикация

Положите локальные артефакты в игнорируемые директории
`squashfs-tools/bin` и `squashfs-tools/lib`, затем выполните:

```bash
./deploy.sh
```

`deploy.sh` копирует файлы в отдельный временный staging-каталог, создаёт
`squashfs-tools.tgz` и загружает его в Sandbox. Дополнительные параметры
передаются в `ya upload`; `--dry-run` позволяет проверить архив без публикации.
После публикации новый id нужно записать в `resources.json`.

Текущий ресурс: `sbr:13473096548`. SHA-256 архива:

```text
48f205d6fd82cdeb92e4c24cdde0afcdee367e0e1733a8b7a64b032e46777c67
```
