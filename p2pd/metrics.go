package main

import (
	psmetrics "github.com/libp2p/go-libp2p-pubsub/metrics"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
)

func enableMetrics() (*prometheus.Exporter, error) {
	opts := prometheus.Options{
		Namespace: "libp2p_daemon",
	}
	pe, err := prometheus.NewExporter(opts)
	if err != nil {
		return nil, err
	}
	view.RegisterExporter(pe)

	if err = psmetrics.Register(); err != nil {
		return nil, err
	}

	return pe, nil
}
