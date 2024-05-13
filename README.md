# otlp-replay
A tool for resending otlp data captured by the [Collector File Expoter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/fileexporter).


# Setup
First you need to generate data with a collector.  If you are using the Collector Contrib and you add the following snippet it will create an uncompressed capture of the otlp:

```yaml
exporters:
  file:
    path: /filedump/otelcol-scc.json
    rotation:
      max_megabytes: 100
      max_backups: 100
```

Once you retrieve the file you can replay it over and over again with:

`otlp-replay -o localhost:4317 otelcol-scc.json`

```
Usage of otlp-replay:
  -o, --output string   The address of the OTLP receiver (default "localhost:4317")
  [args]     []string   The files to parse the data from (default "telemetry.json")
```

