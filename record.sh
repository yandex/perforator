#! /bin/bash
perforator/cmd/cli/perforator record --log-level debug --debug --duration 2m -o ./flame.html --pid $(pidof luajit) &> log.log
chown spar ./flame.html
