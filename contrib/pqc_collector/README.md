# Perfator eBPF Collector

eBPF сборщик данных профилирования для Perforator с XDP программами и Prometheus экспортёром.

## Установка

```bash
# Зависимости
pip install bcc prometheus-client

# Требования к системе
# - Linux ядро 5.10+
# - clang/llvm для компиляции eBPF программ
# - root привилегии для загрузки eBPF
```

## Использование

### Базовое использование

```python
from src.integration.yandex.perforator_ebpf_collector import (
    PerfatorEBpfCollector,
    EbpfConfig,
    PerfMode,
)

# Создание сборщика
collector = PerfatorEBpfCollector(
    config=EbpfConfig(mode=PerfMode.ALL)
)

# Запуск сбора
collector.start()

# ... работа ...

# Остановка сбора
collector.stop()
```

### Сбор данных

```python
# Запуск Prometheus сервера
collector.start_prometheus_server()

# Получение последних событий
events = collector.get_recent_events(limit=100)
for event in events:
    print(f"PID: {event.pid}, Comm: {event.comm}, Type: {event.event_type}")

# Получение flame graph данных
flame_graph = collector.get_flame_graph()
print(f"Root samples: {flame_graph['value']}")
```

### Экспорт данных

```python
# JSON формат
json_data = collector.export_flame_graph_json()

# Collapsed формат (для FlameGraph инструментов)
collapsed = collector.export_flame_graph_collapsed()

# Сохранение в файл
with open("flame_graph.json", "w") as f:
    f.write(json_data)

with open("flame_graph.collapsed", "w") as f:
    f.write(collapsed)
```

### Интеграция с Perforator

```python
from src.integration.yandex.perforator_ebpf_collector import (
    create_perfator_collector,
    PerfMode,
)

# Создание сборщика для Perforator
collector = create_perfator_collector(
    mode=PerfMode.CPU,
    prometheus_port=9090,
    enable_xdp=True,
    enable_kprobes=True,
)

# Запуск
collector.start()
collector.start_prometheus_server()
```

## Конфигурация

### EbpfConfig параметры

| Параметр | По умолчанию | Описание |
|----------|--------------|----------|
| `mode` | `ALL` | Режим профилирования |
| `sample_rate` | `49` | Частота сэмплирования |
| `stack_depth` | `127` | Глубина стека |
| `buffer_size` | `256KB` | Размер буфера |
| `map_size` | `65536` | Размер BPF карты |
| `enable_xdp` | `True` | Включить XDP |
| `enable_kprobes` | `True` | Включить kprobes |
| `enable_uprobes` | `False` | Включить uprobes |
| `prometheus_port` | `9090` | Порт Prometheus |
| `collection_interval` | `1.0` | Интервал сбора (сек) |

### Режимы профилирования

- **CPU**: Анализ потребления CPU
- **MEMORY**: Анализ использования памяти
- **IO**: Анализ ввода-вывода
- **NETWORK**: Анализ сетевого трафика
- **ALL**: Все режимы одновременно

## Архитектура

```
┌─────────────────────────────────────────┐
│        Perfator eBPF Collector          │
├─────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────────┐  │
│  │ XDP Program │  │ Kprobe Handler  │  │
│  │  - Packets  │  │  - Scheduling   │  │
│  │  - Bytes    │  │  - Stack trace  │  │
│  └─────────────┘  └─────────────────┘  │
├─────────────────────────────────────────┤
│           BPF Maps                      │
│  - events (perf event array)           │
│  - stack_traces (stack trace map)      │
│  - flame_graph (hash map)              │
│  - packet_count (hash map)             │
│  - bytes_count (hash map)              │
├─────────────────────────────────────────┤
│        Prometheus Exporter              │
│  - events_total                        │
│  - dropped_events                      │
│  - buffer_usage                        │
│  - collection_duration                 │
│  - packet_count                        │
│  - bytes_transferred                   │
└─────────────────────────────────────────┘
```

## BPF Карты

### events
Тип: `PERF_EVENT_ARRAY`
Хранит события профилирования для передачи в userspace.

### stack_traces
Тип: `STACK_TRACE`
Хранит стеки вызовов для flame graph анализа.

### flame_graph
Тип: `HASH`
Хранит агрегированные данные по стекам вызовов.

### packet_count / bytes_count
Тип: `HASH`
Хранит статистику по процессам (пакеты/байты).

## Метрики Prometheus

| Метрика | Тип | Описание |
|---------|-----|----------|
| `perfator_events_total` | Counter | Всего событий |
| `perfator_dropped_events_total` | Counter | Потерянные события |
| `perfator_buffer_usage_percent` | Gauge | Использование буфера |
| `perfator_collection_duration_seconds` | Histogram | Время сбора |
| `perfator_packets_total` | Counter | Всего пакетов |
| `perfator_bytes_total` | Counter | Всего байт |

## Требования

- Linux ядро 5.10+
- root привилегии
- clang/llvm
- BCC (BPF Compiler Collection)
- Python 3.8+

## Безопасность

- eBPF программы загружаются с проверкой версии ядра
- Ограниченный размер BPF карт для предотвращения OOM
- Автоматическая очистка при остановке
- Без хранения персональных данных

## Лицензия

Apache License 2.0
