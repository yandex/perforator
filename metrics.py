#!/usr/bin/env python

from prometheus_client.parser import text_string_to_metric_families
import json

filename = ".out"

with open(filename, "r") as f:
    data = f.read()

for line in data.splitlines():
    start = "Collected profiler metrics "
    pos = line.find(start)
    if pos != -1:
        metrics = line[pos+len(start):]
        break

metrics_json = json.loads(metrics)
metrics_text = metrics_json["metrics"]

for family in text_string_to_metric_families(metrics_text):
    for sample in family.samples:
        if "lua_" in sample.name:
            print(sample.name, sample.value)
