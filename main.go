package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MultiTelemetry struct {
	ResourceSpans   []ptrace.ResourceSpans `mapstructure:",omitempty"`
	ResourceMetrics []any                  `mapstructure:",omitempty"`
	ResourceLogs    []any                  `mapstructure:",omitempty"`
}

const BufferSize = 10 * 1024 * 1024

var scannerBuffer = make([]byte, BufferSize)

var output = pflag.StringP("output", "o", "localhost:4317", "The address of the OTLP receiver")

func main() {
	pflag.Parse()
	files := pflag.Args()

	clientConn, err := grpc.NewClient(*output, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	tClient := ptraceotlp.NewGRPCClient(clientConn)
	mClient := pmetricotlp.NewGRPCClient(clientConn)
	lClient := plogotlp.NewGRPCClient(clientConn)

	if len(files) == 0 {
		files = []string{"telemetry.json"}
	}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			slog.Error("Failed to open file", "file", file, "error", err)
			continue
		}
		defer f.Close()

		var reader io.Reader = f
		if strings.HasSuffix(file, ".gz") {
			gz, err := gzip.NewReader(f)
			if err != nil {
				slog.Error("Failed to open gzip file", "file", file, "error", err)
				continue
			}
			defer gz.Close()
			reader = gz
		}

		tum := &ptrace.JSONUnmarshaler{}
		mum := &pmetric.JSONUnmarshaler{}
		lum := &plog.JSONUnmarshaler{}
		buf := bufio.NewScanner(reader)
		buf.Buffer(scannerBuffer, BufferSize)

		tcount, mcount, lcount := 0, 0, 0
		for buf.Scan() {
			t, _ := tum.UnmarshalTraces(buf.Bytes())
			if t.SpanCount() > 0 {
				// TODO Send Traces
				tcount++
				_, err := tClient.Export(context.Background(), ptraceotlp.NewExportRequestFromTraces(t))
				if err != nil {
					slog.Error("Failed to send traces", "error", err)
				}
				continue
			}
			m, _ := mum.UnmarshalMetrics(buf.Bytes())
			if m.MetricCount() > 0 {
				mcount++
				_, err := mClient.Export(context.Background(), pmetricotlp.NewExportRequestFromMetrics(m))
				if err != nil {
					slog.Error("Failed to send metrics", "error", err)
				}
				continue
			}
			l, _ := lum.UnmarshalLogs(buf.Bytes())
			if l.LogRecordCount() > 0 {
				lcount++
				_, err := lClient.Export(context.Background(), plogotlp.NewExportRequestFromLogs(l))
				if err != nil {
					slog.Error("Failed to send logs", "error", err)
				}
				continue
			}
			slog.Info("Failed to parse")
		}
		slog.Info("Finished", "traces", tcount, "metrics", mcount, "logs", lcount)
	}
	slog.Info("Done")
}
