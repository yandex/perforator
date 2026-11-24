# Билдер nots

`ya make` запускает команды сборки из [TypeScript bundle nots](../cli/src/builder/).
Билдер готовит зависимости, выполняет `TS_BUILD_SCRIPT` и упаковывает `output.tar`.
В офлайн-сборке `pnpm store add` наполняет store из заранее скачанных архивов
в `__tarballs__`, после чего `pnpm install` использует этот же store.
Тестовые recipes запускаются из того же bundle.

`ckg nodejs-deps` начинает обход графа от узла
`devtools/frontend_build_platform/nots/builder/builder` типа `Bundle`.
Список модулей используется для прогрева кеша. Изменения пути или типа узла
нужно согласовывать с командой CKG и с публикацией графа.

## Сборка из исходников

```shell
ya make -DTS_USE_PREBUILT_NOTS_TOOL=no --host-platform-flag=TS_USE_PREBUILT_NOTS_TOOL=no <target>
```

Опубликованный bundle из [nots.sbr.json](../../../../build/external_resources/nots/nots.sbr.json)
собирает локальный CLI и его зависимости. Остальные модули используют собранный локально bundle.
Для этих флагов также доступны алиасы `-n` и `--no-prebuilt-nots` из [builder.ya.conf](builder.ya.conf).

## Отладка и релиз

`-DTS_LOG=yes` выводит аргументы билдера и команды сборки. Файлы трассировки
можно открыть в [Perfetto](https://ui.perfetto.dev/).

Билдер выпускается вместе с CLI. Проверки, порядок выпуска и откат описаны
в [документации релиза](../cli/docs/maintainers-manual.md).
