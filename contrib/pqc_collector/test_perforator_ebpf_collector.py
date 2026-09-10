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
Тесты для Perforator eBPF сборщика.
"""

import json
import threading
import time
from unittest import TestCase, mock

from src.integration.yandex.perforator_ebpf_collector import (
    CollectionState,
    CollectorStats,
    EbpfConfig,
    FlameGraphNode,
    PerfMode,
    PerfatorEBpfCollector,
    ProfileEvent,
)


class TestEbpfConfig(TestCase):
    """Тесты конфигурации eBPF."""

    def test_default_config(self) -> None:
        """Проверка конфигурации по умолчанию."""
        with mock.patch(
            "src.integration.yandex.perforator_ebpf_collector.HAS_BCC", True
        ):
            config = EbpfConfig()
            self.assertEqual(config.mode, PerfMode.ALL)
            self.assertEqual(config.sample_rate, 49)
            self.assertEqual(config.stack_depth, 127)
            self.assertEqual(config.buffer_size, 256 * 1024)
            self.assertEqual(config.map_size, 65536)
            self.assertTrue(config.enable_xdp)
            self.assertTrue(config.enable_kprobes)
            self.assertFalse(config.enable_uprobes)
            self.assertEqual(config.prometheus_port, 9090)
            self.assertEqual(config.collection_interval, 1.0)

    def test_custom_config(self) -> None:
        """Проверка пользовательской конфигурации."""
        with mock.patch(
            "src.integration.yandex.perforator_ebpf_collector.HAS_BCC", True
        ):
            config = EbpfConfig(
                mode=PerfMode.CPU,
                sample_rate=99,
                stack_depth=63,
                enable_xdp=False,
                prometheus_port=8080,
            )
            self.assertEqual(config.mode, PerfMode.CPU)
            self.assertEqual(config.sample_rate, 99)
            self.assertEqual(config.stack_depth, 63)
            self.assertFalse(config.enable_xdp)
            self.assertEqual(config.prometheus_port, 8080)


class TestFlameGraphNode(TestCase):
    """Тесты узла flame graph."""

    def test_node_creation(self) -> None:
        """Тест создания узла."""
        node = FlameGraphNode(name="test", value=100)
        self.assertEqual(node.name, "test")
        self.assertEqual(node.value, 100)
        self.assertEqual(len(node.children), 0)
        self.assertEqual(node.self_time, 0)
        self.assertEqual(node.total_time, 0)

    def test_node_to_dict(self) -> None:
        """Тест конвертации в словарь."""
        root = FlameGraphNode(name="root", value=200)
        child1 = FlameGraphNode(name="child1", value=100)
        child2 = FlameGraphNode(name="child2", value=100)

        root.children["child1"] = child1
        root.children["child2"] = child2

        result = root.to_dict()

        self.assertEqual(result["name"], "root")
        self.assertEqual(result["value"], 200)
        self.assertEqual(len(result["children"]), 2)
        self.assertIn("child1", [c["name"] for c in result["children"]])
        self.assertIn("child2", [c["name"] for c in result["children"]])

    def test_nested_nodes(self) -> None:
        """Тест вложенных узлов."""
        root = FlameGraphNode(name="root", value=100)
        child = FlameGraphNode(name="child", value=50)
        grandchild = FlameGraphNode(name="grandchild", value=25)

        root.children["child"] = child
        child.children["grandchild"] = grandchild

        result = root.to_dict()

        self.assertEqual(len(result["children"]), 1)
        self.assertEqual(len(result["children"][0]["children"]), 1)


class TestProfileEvent(TestCase):
    """Тесты события профилирования."""

    def test_event_creation(self) -> None:
        """Тест создания события."""
        event = ProfileEvent(
            pid=1234,
            tid=1234,
            comm="test_process",
            timestamp=time.time(),
            event_type="cpu",
            cpu=0,
            duration_ns=1000000,
        )

        self.assertEqual(event.pid, 1234)
        self.assertEqual(event.tid, 1234)
        self.assertEqual(event.comm, "test_process")
        self.assertEqual(event.event_type, "cpu")
        self.assertEqual(event.cpu, 0)
        self.assertEqual(event.duration_ns, 1000000)

    def test_event_with_stack_trace(self) -> None:
        """Тест события со стеком вызовов."""
        stack_trace = [
            "main+0x10",
            "function_a+0x20",
            "function_b+0x30",
        ]

        event = ProfileEvent(
            pid=1234,
            tid=1234,
            comm="test_process",
            timestamp=time.time(),
            event_type="cpu",
            stack_trace=stack_trace,
        )

        self.assertEqual(len(event.stack_trace), 3)
        self.assertEqual(event.stack_trace[0], "main+0x10")


class TestCollectorStats(TestCase):
    """Тесты статистики сборщика."""

    def test_default_stats(self) -> None:
        """Тест статистики по умолчанию."""
        stats = CollectorStats()

        self.assertEqual(stats.total_events, 0)
        self.assertEqual(stats.dropped_events, 0)
        self.assertEqual(stats.sample_count, 0)
        self.assertEqual(stats.active_probes, 0)
        self.assertEqual(stats.buffer_usage_percent, 0.0)
        self.assertEqual(stats.kernel_cpu_time_ms, 0.0)
        self.assertEqual(stats.user_cpu_time_ms, 0.0)
        self.assertEqual(stats.bytes_read, 0)
        self.assertIsNone(stats.last_collection_time)


class TestPerfatorEBpfCollector(TestCase):
    """Тесты eBPF сборщика."""

    def setUp(self) -> None:
        """Настройка тестов."""
        self.config = EbpfConfig(
            mode=PerfMode.ALL,
            enable_xdp=False,
            enable_kprobes=False,
        )

    def test_initialization(self) -> None:
        """Тест инициализации сборщика."""
        collector = PerfatorEBpfCollector(config=self.config)

        self.assertIsNotNone(collector)
        self.assertEqual(collector._state, CollectionState.IDLE)
        self.assertEqual(collector._config.mode, PerfMode.ALL)

    def test_get_stats(self) -> None:
        """Тест получения статистики."""
        collector = PerfatorEBpfCollector(config=self.config)
        stats = collector.get_stats()

        self.assertIsInstance(stats, CollectorStats)
        self.assertEqual(stats.total_events, 0)

    def test_get_recent_events_empty(self) -> None:
        """Тест получения пустого списка событий."""
        collector = PerfatorEBpfCollector(config=self.config)
        events = collector.get_recent_events()

        self.assertEqual(len(events), 0)

    def test_get_flame_graph_empty(self) -> None:
        """Тест получения пустого flame graph."""
        collector = PerfatorEBpfCollector(config=self.config)
        flame_graph = collector.get_flame_graph()

        self.assertEqual(flame_graph["name"], "root")
        self.assertEqual(flame_graph["value"], 0)
        self.assertEqual(len(flame_graph["children"]), 0)

    def test_export_flame_graph_json(self) -> None:
        """Тест экспорта flame graph в JSON."""
        collector = PerfatorEBpfCollector(config=self.config)
        json_str = collector.export_flame_graph_json()

        self.assertIsInstance(json_str, str)

        data = json.loads(json_str)
        self.assertEqual(data["name"], "root")

    def test_export_flame_graph_collapsed(self) -> None:
        """Тест экспорта flame graph в collapsed формат."""
        collector = PerfatorEBpfCollector(config=self.config)
        collapsed = collector.export_flame_graph_collapsed()

        self.assertIsInstance(collapsed, str)
        self.assertEqual(len(collapsed), 0)

    def test_stop_when_not_running(self) -> None:
        """Тест остановки когда сборщик не запущен."""
        collector = PerfatorEBpfCollector(config=self.config)

        collector.stop()

        self.assertEqual(collector._state, CollectionState.IDLE)

    def test_event_buffer_limit(self) -> None:
        """Тест ограничения буфера событий."""
        collector = PerfatorEBpfCollector(config=self.config)

        for i in range(15000):
            event = ProfileEvent(
                pid=i,
                tid=i,
                comm=f"process_{i}",
                timestamp=time.time(),
                event_type="cpu",
            )
            with collector._lock:
                collector._event_buffer.append(event)
                if len(collector._event_buffer) > 10000:
                    collector._event_buffer = collector._event_buffer[-5000:]

        with collector._lock:
            self.assertLessEqual(len(collector._event_buffer), 10000)


class TestPerfMode(TestCase):
    """Тесты режимов профилирования."""

    def test_perf_modes(self) -> None:
        """Тест всех режимов профилирования."""
        self.assertEqual(PerfMode.CPU.value, 1)
        self.assertEqual(PerfMode.MEMORY.value, 2)
        self.assertEqual(PerfMode.IO.value, 3)
        self.assertEqual(PerfMode.NETWORK.value, 4)
        self.assertEqual(PerfMode.ALL.value, 5)


if __name__ == "__main__":
    import unittest

    unittest.main()
