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
Бенчмарк для Perforator eBPF сборщика.

Замеряет производительность eBPF операций и overhead сбора данных.
"""

import statistics
import time
from typing import Optional

from src.integration.yandex.perforator_ebpf_collector import (
    EbpfConfig,
    FlameGraphNode,
    PerfMode,
    PerfatorEBpfCollector,
    ProfileEvent,
)


def benchmark_flame_graph_operations(
    iterations: int = 1000,
) -> dict[str, float]:
    """Бенчмарк операций flame graph."""
    root = FlameGraphNode(name="root", value=0)

    add_times = []
    to_dict_times = []

    for i in range(iterations):
        stack = [
            f"function_{j}" for j in range(10)
        ]

        start = time.perf_counter()
        node = root
        for frame in reversed(stack):
            if frame not in node.children:
                node.children[frame] = FlameGraphNode(
                    name=frame, value=0
                )
            node = node.children[frame]
            node.value += 1
        add_times.append(time.perf_counter() - start)

        start = time.perf_counter()
        _ = root.to_dict()
        to_dict_times.append(time.perf_counter() - start)

    return {
        "flame_graph_add_avg_us": statistics.mean(add_times) * 1_000_000,
        "flame_graph_add_p95_us": sorted(add_times)[int(0.95 * len(add_times))] * 1_000_000,
        "flame_graph_to_dict_avg_us": statistics.mean(to_dict_times) * 1_000_000,
        "flame_graph_to_dict_p95_us": sorted(to_dict_times)[int(0.95 * len(to_dict_times))] * 1_000_000,
    }


def benchmark_event_processing(
    iterations: int = 10000,
) -> dict[str, float]:
    """Бенчмарк обработки событий."""
    creation_times = []
    buffer_append_times = []

    buffer: list[ProfileEvent] = []

    for i in range(iterations):
        start = time.perf_counter()
        event = ProfileEvent(
            pid=i % 1000,
            tid=i % 1000,
            comm=f"process_{i % 100}",
            timestamp=time.time(),
            event_type="cpu",
            cpu=i % 8,
            duration_ns=i * 1000,
        )
        creation_times.append(time.perf_counter() - start)

        start = time.perf_counter()
        buffer.append(event)
        if len(buffer) > 10000:
            buffer = buffer[-5000:]
        buffer_append_times.append(time.perf_counter() - start)

    return {
        "event_creation_avg_us": statistics.mean(creation_times) * 1_000_000,
        "event_creation_p95_us": sorted(creation_times)[int(0.95 * len(creation_times))] * 1_000_000,
        "buffer_append_avg_us": statistics.mean(buffer_append_times) * 1_000_000,
        "buffer_append_p95_us": sorted(buffer_append_times)[int(0.95 * len(buffer_append_times))] * 1_000_000,
    }


def benchmark_export_operations(
    iterations: int = 100,
) -> dict[str, float]:
    """Бенчмарк операций экспорта."""
    root = FlameGraphNode(name="root", value=0)

    for i in range(1000):
        stack = [f"func_{j}" for j in range(20)]
        node = root
        for frame in reversed(stack):
            if frame not in node.children:
                node.children[frame] = FlameGraphNode(
                    name=frame, value=0
                )
            node = node.children[frame]
            node.value += 1

    json_times = []
    collapsed_times = []

    import json

    for _ in range(iterations):
        start = time.perf_counter()
        json.dumps(root.to_dict(), indent=2)
        json_times.append(time.perf_counter() - start)

        start = time.perf_counter()
        lines = []
        stack = []

        def flatten(node: FlameGraphNode, current_stack: list[str]) -> None:
            current_stack = current_stack + [node.name]
            if not node.children:
                stack_str = ";".join(reversed(current_stack))
                lines.append(f"{stack_str} {node.value}")
            else:
                for child in node.children.values():
                    flatten(child, current_stack)

        flatten(root, [])
        "\n".join(lines)
        collapsed_times.append(time.perf_counter() - start)

    return {
        "export_json_avg_ms": statistics.mean(json_times) * 1000,
        "export_json_p95_ms": sorted(json_times)[int(0.95 * len(json_times))] * 1000,
        "export_collapsed_avg_ms": statistics.mean(collapsed_times) * 1000,
        "export_collapsed_p95_ms": sorted(collapsed_times)[int(0.95 * len(collapsed_times))] * 1000,
    }


def benchmark_memory_usage() -> dict[str, int]:
    """Замер использования памяти."""
    import sys

    root = FlameGraphNode(name="root", value=0)

    for i in range(10000):
        stack = [f"function_{j}" for j in range(15)]
        node = root
        for frame in reversed(stack):
            if frame not in node.children:
                node.children[frame] = FlameGraphNode(
                    name=frame, value=0
                )
            node = node.children[frame]
            node.value += 1

    def count_nodes(node: FlameGraphNode) -> int:
        count = 1
        for child in node.children.values():
            count += count_nodes(child)
        return count

    node_count = count_nodes(root)

    return {
        "flame_graph_nodes": node_count,
        "flame_graph_bytes_estimate": node_count * 200,
    }


def benchmark_concurrent_access(
    iterations: int = 1000,
    num_threads: int = 4,
) -> dict[str, float]:
    """Бенчмарк конкурентного доступа."""
    import threading

    root = FlameGraphNode(name="root", value=0)
    lock = threading.Lock()
    errors = []

    def worker(thread_id: int) -> None:
        try:
            for i in range(iterations):
                stack = [f"thread_{thread_id}_func_{j}" for j in range(5)]
                with lock:
                    node = root
                    for frame in reversed(stack):
                        if frame not in node.children:
                            node.children[frame] = FlameGraphNode(
                                name=frame, value=0
                            )
                        node = node.children[frame]
                        node.value += 1
        except Exception as e:
            errors.append(e)

    start = time.perf_counter()
    threads = []
    for i in range(num_threads):
        t = threading.Thread(target=worker, args=(i,))
        threads.append(t)
        t.start()

    for t in threads:
        t.join()

    duration = time.perf_counter() - start

    return {
        "concurrent_duration_ms": duration * 1000,
        "operations_per_ms": (iterations * num_threads) / (duration * 1000),
        "errors": len(errors),
    }


def run_all_benchmarks() -> None:
    """Запуск всех бенчмарков."""
    print("=" * 60)
    print("Perforator eBPF Collector Benchmarks")
    print("=" * 60)

    print("\n1. Flame Graph Operations:")
    fg_results = benchmark_flame_graph_operations()
    for key, value in fg_results.items():
        print(f"   {key}: {value:.3f}us")

    print("\n2. Event Processing:")
    event_results = benchmark_event_processing()
    for key, value in event_results.items():
        print(f"   {key}: {value:.3f}us")

    print("\n3. Export Operations:")
    export_results = benchmark_export_operations()
    for key, value in export_results.items():
        print(f"   {key}: {value:.3f}ms")

    print("\n4. Memory Usage:")
    memory_results = benchmark_memory_usage()
    for key, value in memory_results.items():
        print(f"   {key}: {value}")

    print("\n5. Concurrent Access:")
    concurrent_results = benchmark_concurrent_access()
    for key, value in concurrent_results.items():
        print(f"   {key}: {value:.3f}" if isinstance(value, float) else f"   {key}: {value}")

    print("\n" + "=" * 60)
    print("Benchmarks completed")
    print("=" * 60)


if __name__ == "__main__":
    run_all_benchmarks()
