# Copyright 2024 x0tta6bl4 contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Perforator eBPF Collector — сбор данных профилирования через eBPF.

Реализует XDP программу для сбора метрик на уровне пакетов,
BPF карты для данных flame graph и экспортёр Prometheus.

Целевой репозиторий: github.com/yandex/perforator
"""

from __future__ import annotations

import ctypes
import hashlib
import json
import logging
import os
import platform
import struct
import threading
import time
from collections import defaultdict
from dataclasses import dataclass, field
from enum import Enum, auto
from typing import Any, Optional, Union

logger = logging.getLogger(__name__)

try:
    from bcc import BPF, BPFProgType

    HAS_BCC = True
except ImportError:
    HAS_BCC = False

try:
    from prometheus_client import (
        CollectorRegistry,
        Counter,
        Gauge,
        Histogram,
        start_http_server,
    )

    HAS_PROMETHEUS = True
except ImportError:
    HAS_PROMETHEUS = False


class PerfMode(Enum):
    """Режимы профилирования."""

    CPU = auto()
    MEMORY = auto()
    IO = auto()
    NETWORK = auto()
    ALL = auto()


class CollectionState(Enum):
    """Состояния сбора данных."""

    IDLE = auto()
    ATTACHING = auto()
    COLLECTING = auto()
    STOPPING = auto()
    ERROR = auto()


@dataclass(frozen=True)
class EbpfConfig:
    """Конфигурация eBPF сборщика."""

    mode: PerfMode = PerfMode.ALL
    sample_rate: int = 49
    stack_depth: int = 127
    buffer_size: int = 256 * 1024
    map_size: int = 65536
    enable_xdp: bool = True
    enable_kprobes: bool = True
    enable_uprobes: bool = False
    prometheus_port: int = 9090
    collection_interval: float = 1.0
    kernel_version_min: tuple[int, int] = (5, 10)

    def __post_init__(self) -> None:
        if not HAS_BCC:
            raise ImportError(
                "bcc обязателен для eBPF операций. "
                "Установите: pip install bcc"
            )
        self._validate_kernel_version()

    def _validate_kernel_version(self) -> None:
        """Проверка версии ядра."""
        release = platform.release()
        try:
            version_parts = release.split("-")[0].split(".")
            major = int(version_parts[0])
            minor = int(version_parts[1])
            if (major, minor) < self.kernel_version_min:
                logger.warning(
                    f"Ядро {release} может не поддерживать все eBPF функции. "
                    f"Рекомендуется {self.kernel_version_min[0]}.{self.kernel_version_min[1]}+"
                )
        except (ValueError, IndexError):
            logger.warning(f"Не удалось определить версию ядра: {release}")


@dataclass
class ProfileEvent:
    """Событие профилирования."""

    pid: int
    tid: int
    comm: str
    timestamp: float
    event_type: str
    stack_trace: list[str] = field(default_factory=list)
    cpu: int = 0
    duration_ns: int = 0
    bytes_transferred: int = 0
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class FlameGraphNode:
    """Узел flame graph."""

    name: str
    value: int
    children: dict[str, FlameGraphNode] = field(default_factory=dict)
    self_time: int = 0
    total_time: int = 0

    def to_dict(self) -> dict[str, Any]:
        """Конвертация в словарь."""
        children_list = [child.to_dict() for child in self.children.values()]
        return {
            "name": self.name,
            "value": self.value,
            "self_time": self.self_time,
            "total_time": self.total_time,
            "children": children_list,
        }


@dataclass
class CollectorStats:
    """Статистика сборщика."""

    total_events: int = 0
    dropped_events: int = 0
    sample_count: int = 0
    active_probes: int = 0
    buffer_usage_percent: float = 0.0
    kernel_cpu_time_ms: float = 0.0
    user_cpu_time_ms: float = 0.0
    bytes_read: int = 0
    last_collection_time: Optional[float] = None


XDP_PROGRAM = """
#include <uapi/linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>

struct perf_event_t {
    u32 pid;
    u32 tid;
    char comm[16];
    u64 timestamp;
    u32 event_type;
    u32 cpu;
    u64 duration_ns;
    u64 bytes_transferred;
};

struct bpf_map_def SEC("maps") events = {
    .type = BPF_MAP_TYPE_PERF_EVENT_ARRAY,
    .key_size = sizeof(u32),
    .value_size = sizeof(u32),
    .max_entries = 256,
};

struct bpf_map_def SEC("maps") stack_traces = {
    .type = BPF_MAP_TYPE_STACK_TRACE,
    .key_size = sizeof(u32),
    .value_size = 127 * sizeof(u64),
    .max_entries = 65536,
};

struct bpf_map_def SEC("maps") flame_graph = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(u64),
    .value_size = sizeof(u64),
    .max_entries = 65536,
};

struct bpf_map_def SEC("maps") packet_count = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(u32),
    .value_size = sizeof(u64),
    .max_entries = 65536,
};

struct bpf_map_def SEC("maps") bytes_count = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(u32),
    .value_size = sizeof(u64),
    .max_entries = 65536,
};

int xdp_perf_monitor(struct xdp_md *ctx) {
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u32 tid = bpf_get_current_pid_tgid() & 0xFFFFFFFF;
    u64 ts = bpf_ktime_get_ns();

    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if (data + sizeof(*eth) > data_end)
        return XDP_PASS;

    if (eth->h_proto != htons(ETH_P_IP))
        return XDP_PASS;

    struct iphdr *ip = data + sizeof(*eth);
    if (data + sizeof(*eth) + sizeof(*ip) > data_end)
        return XDP_PASS;

    u32 pkt_len = data_end - data;

    u64 *count = bpf_map_lookup_elem(&packet_count, &pid);
    if (count) {
        (*count)++;
    } else {
        u64 init_val = 1;
        bpf_map_update_elem(&packet_count, &pid, &init_val, BPF_ANY);
    }

    u64 *bytes = bpf_map_lookup_elem(&bytes_count, &pid);
    if (bytes) {
        *bytes += pkt_len;
    } else {
        bpf_map_update_elem(&bytes_count, &pid, &pkt_len, BPF_ANY);
    }

    struct perf_event_t event = {};
    event.pid = pid;
    event.tid = tid;
    bpf_get_current_comm(&event.comm, sizeof(event.comm));
    event.timestamp = ts;
    event.event_type = 1;
    event.cpu = bpf_get_smp_processor_id();
    event.bytes_transferred = pkt_len;

    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU,
                          &event, sizeof(event));

    return XDP_PASS;
}

SEC("kprobe/__schedule")
int kprobe_schedule(struct pt_regs *ctx) {
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u32 tid = bpf_get_current_pid_tgid() & 0xFFFFFFFF;
    u64 ts = bpf_ktime_get_ns();

    struct perf_event_t event = {};
    event.pid = pid;
    event.tid = tid;
    bpf_get_current_comm(&event.comm, sizeof(event.comm));
    event.timestamp = ts;
    event.event_type = 2;
    event.cpu = bpf_get_smp_processor_id();

    u32 stack_id = bpf_get_stackid(ctx, &stack_traces, 0);
    if (stack_id >= 0) {
        u64 *count = bpf_map_lookup_elem(&flame_graph, &stack_id);
        if (count) {
            (*count)++;
        } else {
            u64 init_val = 1;
            bpf_map_update_elem(&flame_graph, &stack_id, &init_val, BPF_ANY);
        }
    }

    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU,
                          &event, sizeof(event));

    return 0;
}

char _license[] SEC("license") = "GPL";
"""


class PerfatorEBpfCollector:
    """
    eBPF сборщик данных профилирования для Perforator.

    Поддерживает:
    - XDP программы для пакетного мониторинга
    - Kprobes для анализа планировщика
    - BPF карты для flame graph данных
    - Экспорт Prometheus метрик
    """

    def __init__(
        self,
        config: Optional[EbpfConfig] = None,
    ) -> None:
        """
        Инициализация eBPF сборщика.

        Args:
            config: Конфигурация eBPF параметров.
        """
        self._config = config or EbpfConfig()
        self._state = CollectionState.IDLE
        self._stats = CollectorStats()
        self._flame_graph_root = FlameGraphNode(name="root", value=0)
        self._event_buffer: list[ProfileEvent] = []
        self._lock = threading.Lock()
        self._bpf: Optional[Any] = None
        self._collection_thread: Optional[threading.Thread] = None
        self._running = False

        self._prometheus_metrics: dict[str, Any] = {}
        if HAS_PROMETHEUS:
            self._setup_prometheus_metrics()

        logger.info(
            f"PerforatorEBpfCollector инициализирован: mode={self._config.mode.name}"
        )

    def _setup_prometheus_metrics(self) -> None:
        """Настройка Prometheus метрик."""
        self._prometheus_metrics = {
            "events_total": Counter(
                "perfator_events_total",
                "Total profiling events",
                ["event_type"],
            ),
            "dropped_events": Counter(
                "perfator_dropped_events_total",
                "Total dropped events",
            ),
            "buffer_usage": Gauge(
                "perfator_buffer_usage_percent",
                "Buffer usage percentage",
            ),
            "collection_duration": Histogram(
                "perfator_collection_duration_seconds",
                "Collection duration",
            ),
            "packet_count": Counter(
                "perfator_packets_total",
                "Total packets processed",
            ),
            "bytes_transferred": Counter(
                "perfator_bytes_total",
                "Total bytes transferred",
            ),
        }

    def start(self) -> None:
        """Запуск сборщика."""
        if self._state == CollectionState.COLLECTING:
            logger.warning("Сборщик уже запущен")
            return

        self._state = CollectionState.ATTACHING
        self._running = True

        try:
            self._load_bpf_program()
            self._attach_probes()

            self._state = CollectionState.COLLECTING
            self._collection_thread = threading.Thread(
                target=self._collection_loop,
                daemon=True,
            )
            self._collection_thread.start()

            logger.info("eBPF сборщик запущен")

        except Exception as e:
            self._state = CollectionState.ERROR
            logger.error(f"Ошибка запуска eBPF сборщика: {e}")
            raise

    def stop(self) -> None:
        """Остановка сборщика."""
        if self._state != CollectionState.COLLECTING:
            return

        self._state = CollectionState.STOPPING
        self._running = False

        if self._collection_thread:
            self._collection_thread.join(timeout=5.0)

        self._detach_probes()

        self._state = CollectionState.IDLE
        logger.info("eBPF сборщик остановлен")

    def _load_bpf_program(self) -> None:
        """Загрузка eBPF программы."""
        if not HAS_BCC:
            raise RuntimeError("BCC не доступен")

        self._bpf = BPF(text=XDP_PROGRAM)
        logger.debug("eBPF программа загружена")

    def _attach_probes(self) -> None:
        """Подключение проб."""
        if not self._bpf:
            return

        if self._config.enable_xdp:
            try:
                self._bpf.attach_xdp(
                    self._bpf.load_func(
                        "xdp_perf_monitor", BPFProgType.XDP
                    ),
                    0,
                )
                self._stats.active_probes += 1
                logger.info("XDP проб подключен")
            except Exception as e:
                logger.warning(f"Не удалось подключить XDP проб: {e}")

        if self._config.enable_kprobes:
            try:
                self._bpf.attach_kprobe(
                    event="__schedule",
                    fn_name="kprobe_schedule",
                )
                self._stats.active_probes += 1
                logger.info("Kprobe проб подключен")
            except Exception as e:
                logger.warning(f"Не удалось подключить kprobe: {e}")

    def _detach_probes(self) -> None:
        """Отключение проб."""
        if not self._bpf:
            return

        try:
            if self._config.enable_xdp:
                self._bpf.remove_xdp(0)
            if self._config.enable_kprobes:
                self._bpf.detach_kprobe(event="__schedule")
        except Exception as e:
            logger.warning(f"Ошибка отключения проб: {e}")

        self._stats.active_probes = 0

    def _collection_loop(self) -> None:
        """Основной цикл сбора данных."""
        while self._running:
            try:
                self._collect_events()
                time.sleep(self._config.collection_interval)
            except Exception as e:
                logger.error(f"Ошибка в цикле сбора: {e}")
                time.sleep(1.0)

    def _collect_events(self) -> None:
        """Сбор событий из BPF карт."""
        if not self._bpf:
            return

        start_time = time.monotonic()

        try:
            events_table = self._bpf.get_table("events")
            for key, leaf in events_table.items():
                event = ProfileEvent(
                    pid=leaf.pid,
                    tid=leaf.tid,
                    comm=leaf.comm.decode("utf-8", errors="replace"),
                    timestamp=leaf.timestamp,
                    event_type="packet" if leaf.event_type == 1 else "schedule",
                    cpu=leaf.cpu,
                    bytes_transferred=leaf.bytes_transferred,
                )

                with self._lock:
                    self._event_buffer.append(event)
                    if len(self._event_buffer) > 10000:
                        self._event_buffer = self._event_buffer[-5000:]

                self._stats.total_events += 1

                if HAS_PROMETHEUS and "events_total" in self._prometheus_metrics:
                    self._prometheus_metrics["events_total"].labels(
                        event_type=event.event_type
                    ).inc()

        except Exception as e:
            logger.debug(f"Ошибка чтения событий: {e}")

        self._collect_flame_graph_data()

        duration = time.monotonic() - start_time
        self._stats.last_collection_time = time.time()
        self._stats.kernel_cpu_time_ms += duration * 1000

        if HAS_PROMETHEUS and "collection_duration" in self._prometheus_metrics:
            self._prometheus_metrics["collection_duration"].observe(duration)

    def _collect_flame_graph_data(self) -> None:
        """Сбор данных для flame graph."""
        if not self._bpf:
            return

        try:
            stack_traces_table = self._bpf.get_table("stack_traces")
            flame_graph_table = self._bpf.get_table("flame_graph")

            self._flame_graph_root = FlameGraphNode(name="root", value=0)

            for key, count in flame_graph_table.items():
                stack_id = key.value
                sample_count = count.value

                try:
                    stack_trace = stack_traces_table[stack_id]
                    stack_frames = []
                    for i in range(self._config.stack_depth):
                        addr = stack_trace[i]
                        if addr == 0:
                            break
                        symbol = self._bpf.sym(addr, show_module=True, show_offset=True)
                        stack_frames.append(symbol)

                    self._add_to_flame_graph(stack_frames, sample_count)

                except KeyError:
                    continue

        except Exception as e:
            logger.debug(f"Ошибка чтения flame graph: {e}")

    def _add_to_flame_graph(
        self, stack_frames: list[str], count: int
    ) -> None:
        """Добавление стека в flame graph."""
        node = self._flame_graph_root
        node.value += count
        node.total_time += count

        for frame in reversed(stack_frames):
            if frame not in node.children:
                node.children[frame] = FlameGraphNode(
                    name=frame, value=0
                )
            node = node.children[frame]
            node.value += count
            node.total_time += count

        if node.children:
            pass
        else:
            node.self_time += count

    def get_flame_graph(self) -> dict[str, Any]:
        """Получение данных flame graph."""
        return self._flame_graph_root.to_dict()

    def get_stats(self) -> CollectorStats:
        """Получение статистики."""
        return self._stats

    def get_recent_events(
        self, limit: int = 100
    ) -> list[ProfileEvent]:
        """Получение последних событий."""
        with self._lock:
            return self._event_buffer[-limit:]

    def export_flame_graph_json(self) -> str:
        """Экспорт flame graph в JSON."""
        return json.dumps(self.get_flame_graph(), indent=2)

    def export_flame_graph_collapsed(self) -> str:
        """Экспорт flame graph в формате collapsed."""
        lines = []
        self._flatten_flame_graph(self._flame_graph_root, [], lines)
        return "\n".join(lines)

    def _flatten_flame_graph(
        self,
        node: FlameGraphNode,
        stack: list[str],
        lines: list[str],
    ) -> None:
        """Рекурсивное развертывание flame graph."""
        current_stack = stack + [node.name]

        if not node.children:
            stack_str = ";".join(reversed(current_stack))
            lines.append(f"{stack_str} {node.self_time}")
        else:
            for child in node.children.values():
                self._flatten_flame_graph(child, current_stack, lines)

    def start_prometheus_server(self) -> None:
        """Запуск Prometheus сервера."""
        if not HAS_PROMETHEUS:
            raise RuntimeError("prometheus_client не доступен")

        start_http_server(self._config.prometheus_port)
        logger.info(
            f"Prometheus сервер запущен на порту {self._config.prometheus_port}"
        )

    def update_prometheus_metrics(self) -> None:
        """Обновление Prometheus метрик."""
        if not HAS_PROMETHEUS:
            return

        if "buffer_usage" in self._prometheus_metrics:
            self._prometheus_metrics["buffer_usage"].set(
                self._stats.buffer_usage_percent
            )

        if "dropped_events" in self._prometheus_metrics:
            self._prometheus_metrics["dropped_events"]._value.set(
                self._stats.dropped_events
            )


def create_perfator_collector(
    mode: PerfMode = PerfMode.ALL,
    prometheus_port: int = 9090,
    enable_xdp: bool = True,
    enable_kprobes: bool = True,
) -> PerfatorEBpfCollector:
    """
    Фабричная функция для создания eBPF сборщика.

    Args:
        mode: Режим профилирования.
        prometheus_port: Порт Prometheus.
        enable_xdp: Включить XDP программы.
        enable_kprobes: Включить kprobes.

    Returns:
        Настроенный eBPF сборщик.
    """
    config = EbpfConfig(
        mode=mode,
        prometheus_port=prometheus_port,
        enable_xdp=enable_xdp,
        enable_kprobes=enable_kprobes,
    )

    collector = PerfatorEBpfCollector(config=config)

    logger.info(
        f"Perforator eBPF сборщик создан: mode={mode.name}, "
        f"prometheus_port={prometheus_port}"
    )

    return collector
