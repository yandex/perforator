# nots builder

Внутренний инструмент, выполняющий сборку Frontend-проектов (узлов сборки) в рамках сборщика `ya make`.

## Собирает папку `node_modules`

Запускает `pnpm install`, используя скаченные заранее архивы пакетов.

В результате ожидается архив `node_modules.tar`.

_Скоро выпилим_

## Запускает сборщики типа `tsc`, `webpack`, `next`, `vite`

В соответствии с типом таргета (`TS_TSC`, `TS_WEBPACK`, `TS_NEXT`, `TS_VITE` макросами в `ya.make`) для непосредственной сборки проекта.

Пакеты должны быть установлены в качестве зависимостей проекта, обычно в `devDependencies`.

В результате ожидается архив `<module_name>.output.tar`.

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

Есть возможность сгенерировать файлы с трейсами для отладки производительности на всех этапах сборки (внутри `builder`)

Для этого нужно запустить `ya make` с опцией `-DTS_TRACE` (т.е. установкой переменной `TS_TRACE=yes`).

В выходной архив добавится папка `.traces` с файлом в `Trace Event Format`, который можно открыть в Chrome Devtools или в https://ui.perfetto.dev/.

Подробнее, о том, как читать трейсинг тут: https://a.yandex-team.ru/arcadia/devtools/frontend_build_platform/libraries/logging/README.md

После запуска из `ya tool nots build` архивы распакуются и трейсы следует искать по таким путям:

- Для `create-node-modules`: `./node_modules/.traces/builder_create-node-modules.trace.json`
- Для прочих команд: `./.traces/builder_<command>.trace.json`
