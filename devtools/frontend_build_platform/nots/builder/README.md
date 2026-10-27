# nots builder

Внутренний инструмент, выполняющий сборку Frontend-проектов (узлов сборки) в рамках сборщика `ya make`.

## Собирает папку `node_modules`

Запускает `pnpm install`, используя скаченные заранее архивы пакетов.

В результате ожидается архив `node_modules.tar`.

_Скоро выпилим_

## Запускает сборщики типа `tsc`, `webpack`, `rspack`, `next`, `vite`

Команда сборки задаётся через `TS_BUILD_SCRIPT` в `TS_LIBRARY`. Для оставшихся legacy-модулей `TS_TSC`, `TS_WEBPACK` и `TS_VITE` команда определяется типом таргета.

Пакеты должны быть установлены в качестве зависимостей проекта, обычно в `devDependencies`.

В результате ожидается архив `output.tar`.

## Релиз

`builder` собирается в бинарники, которые используются внутри сборки уже собранными.
Маппинг собранных под разные платформы ресурсы описан в [resources.json](./resources.json).

Временно релиз настроен на ручной запуск, причем в двух местах: при запуске релиза и при влитии результата в транк.

При изменениях в builder в рамках PR нужно:

0. Нужно пошарить PR на `robot-nots` **ВАЖНО!**
1. Запустить ручной action `devtools/frontend_build_platform/nots/builder: Build nots-builder (from PR)`
2. Будет собран builder из ветки и новые ресурсы будут прописаны в [resources.json](./resources.json)
3. С новым коммитом будет перезапущены все tier0 проекты на TS макросах - проверка, что изменения билдера ничего не сломали
4. "Релиз" новой версии - это влитие ветки.

{% note warning "ПОМНИ!" %}

При дополнительных коммитах в PR нужно вручную запускать action

```
devtools/frontend_build_platform/nots/builder: Build nots-builder (from PR)
```

{% endnote %}

Можно не делать проверку в рамках своего ПР, а после влития запустить
[релиз](https://a.yandex-team.ru/projects/?frontend_build_platform/ci/releases/timeline?dir=devtools%2Ffrontend_build_platform%2Fnots%2Fbuilder&id=release-nots),
который:

1. Соберет новые ресурсы
2. Обновит [resources.json](./resources.json)
3. Сделает PR в транк
4. Будет ждать влития

## Собрать и обновить бинарники локально, без CI

Из текущего checkout можно собрать все четыре платформы, загрузить архивы в Sandbox
и обновить локальный `resources.json`. Из корня Аркадии:

```shell
bash devtools/frontend_build_platform/nots/builder/build_resources.sh
```

Скрипт запускает `ya package` с описанием [ya_package.json](./ya_package.json).
Все четыре платформы, release-режим и флаги `TS_USE_PREBUILT_NOTS_TOOL=no`, `STRIP=yes`
заданы в этом файле. Пакет содержит `linux/builder`, `linux-aarch64/builder`,
`darwin/builder` и `darwin-arm64/builder`.
Затем скрипт переупаковывает каждый бинарник в отдельный `builder.tgz`
с файлом `builder` в корне, как в сборке из `a.yaml`.
Ресурсы получают тип `NOTS_BUILDER_BINARY`, владельца `FRONTEND_BUILD_PLATFORM`,
`ttl=inf`, атрибут `platform` и сведения об исходной ревизии и checkout.

Чтобы только собрать пакет без загрузки, из директории `nots/builder`:

```shell
ya package ya_package.json
```

По умолчанию сборка локальная. Результат — `nots-builder.<revision>.tar.gz`.

`resources.json` заменяется после успешной сборки и загрузки всех платформ.
Архивы и ответы загрузки остаются во временной директории, путь к ней выводится в начале.
Скрипт не создаёт коммитов, не отправляет изменения и не запускает CI.

## Локальный запуск

Чтобы отключить запуск предсобаранного бинарника и запускать с учетом изменений в ветке, то нужно указать переменную:

```shell
ya make -DTS_USE_PREBUILT_NOTS_TOOL=no --host-platform-flag=TS_USE_PREBUILT_NOTS_TOOL=no
```

Для удобства можно [настроить алиас](https://docs.yandex-team.ru/yatool/usage/options#primer-otklyuchenie-predpostroennyh-tulov) в `junk/<username>/ya.conf`:

```toml
[[include]]
path = "${YA_ARCADIA_ROOT}/devtools/frontend_build_platform/nots/builder/builder.ya.conf"
```

После чего будут доступны алиасы `-n` и `--no-prebuilt-nots`:

```shell
ya make -n
ya make --no-prebuilt-nots
```

## Запуск

Обычно запускается внутри сборки `ya make`, как команда (`.CMD`).

Ручной запуск сложен, т.к. требует определенный набор переданных в аргументах подготовленных директорий.
Если очень уж хочется, то можно бросить исключение из builder (упасть).
Тогда сборка завершится с ошибкой, а при этом сборочные папки не удаляются, в лог пишется команда запуска builder со всем аргументами.
Таким образом его можно поправить и перезапустить с теми же параметрами в режиме дебага, например.

## Отладка

Можно включить логирование с опцией `-DTS_LOG=yes`, тогда `builder` напишет в консоль переданные аргументы, то, как он их распарсил.

А также покажет команду для запуска самого сборщика (можно будет скопировать и выполнить самому).

### Отладка с брекпоинтами

Настоящий дебаг в IDE делается так:

1. Настраивается окружение с помощью команды `ya ide venv` из папки `cli` (в принципе полезно, в codenv должно быть уже настроено);
2. Это окружение активируется в IDE;
3. Запустить `builder` с опцией `-DTS_LOG`, скопировать все аргументы командной строки из выхлопа;
4. Настроить запуск `main.py` в IDE в режиме дебага и передать все эти аргументы.

## Профилирование

В папке `~/.nots/traces/<MODDIR>` складываются файлы с трейсами в формате `Trace Event Format`.

Для каждой команды есть отдельный файл, который можно открыть в Chrome Devtools или в https://ui.perfetto.dev/.

Подробнее, о том, как читать трейсинг [тут](../../libraries/logging/README.md).
