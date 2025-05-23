# Operations with local profile

## Create flamegraph from local profile

Create a flamegraph inside HTML from (un)compressed pprof file.

```console
perforator profile flamegraph pprof -i ~/symbolized_profile.pprof > symbolized.html
```

## Symbolize pprof using local binary with debug info

```console
perforator symbolize local --binary BINARY1,BINARY2 unsymbolized_profile.pprof
```
