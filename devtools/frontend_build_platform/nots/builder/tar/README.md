# nots-tar

Встроенный инструмент `nots/builder` для упаковки и распаковки артефактов сборки и слоя `node_modules`.

| Команда | Что делает |
|---|---|
| `nots-tar pack-tar OUTPUT` | Создаёт GNU tar без сжатия. Билдер использует этот режим для `output.tar` и `workspace_node_modules.tar`. |
| `nots-tar pack-layer OUTPUT` | Создаёт GNU tar со сжатием Zstd уровня 1 для слоя `node_modules`. |
| `nots-tar unpack-tar ARCHIVE DIRECTORY PREFIX [--keep-existing]` | Распаковывает обычный tar в существующий каталог. Удаляет `PREFIX` из путей записей и целей hardlink’ов. Для архива без префикса передайте `''`. При `--keep-existing` сохраняет записи, уже существующие на диске. |
| `nots-tar unpack-layer ARCHIVE NEW_DIRECTORY` | Распаковывает слой в новый каталог. Родительский каталог должен существовать. Сохраняет содержимое, права, симлинки и hardlink’и. |

Обе команды распаковки записывают независимые файлы параллельно. Если записи обычного tar конфликтуют по путям или зависят от порядка, инструмент применяет их в порядке архива.

Обе команды упаковки читают из stdin пары «путь на диске, путь в архиве». Каждое поле, включая последнее, заканчивается байтом NUL. Для каталога путь в архиве задаёт префикс его содержимого. Префикс `.` помещает содержимое в корень архива. Записи идут в фиксированном порядке с `mtime=0`. Билдер нормализует метаданные pnpm перед упаковкой слоя.

```sh
printf '%s\000' ./dist dist ./package.json package.json | nots-tar pack-tar output.tar
printf '%s\000' ./node_modules . | nots-tar pack-layer node_modules.layer
nots-tar unpack-layer node_modules.layer ./restored-node_modules
mkdir -p ./restored-output
nots-tar unpack-tar output.tar ./restored-output '' --keep-existing
```

[Обработка команд](main.cpp#L739) · [Вызов из билдера](../api/tar_tool.py#L33)
