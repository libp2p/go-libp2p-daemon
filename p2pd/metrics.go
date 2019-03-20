package main

import (
	"log"
	"strings"

	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
)

var metricsModules = map[string][]*view.View{}

func metricsModuleNames() string {
	moduleNames := make([]string, 0, len(metricsModules))
	for module := range metricsModules {
		moduleNames = append(moduleNames, module)
	}
	return strings.Join(moduleNames, ",")
}

func enableMetrics(moduleNames []string) (*prometheus.Exporter, error) {
	opts := prometheus.Options{}
	pe, err := prometheus.NewExporter(opts)
	if err != nil {
		return nil, err
	}
	view.RegisterExporter(pe)

	for _, name := range moduleNames {
		if views, ok := metricsModules[name]; ok {
			view.Register(views...)
		} else {
			log.Println("couldn't find metrics for module by name:", name)
		}
	}

	return pe, nil
}
