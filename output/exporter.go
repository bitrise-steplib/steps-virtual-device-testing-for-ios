package output

import (
	"github.com/bitrise-io/go-steputils/v2/export"
)

type OutputExporter interface {
	ExportOutput(key, value string) error
}

type outputExporter struct {
	exporter export.Exporter
}

func NewOutputExporter(exporter export.Exporter) OutputExporter {
	return &outputExporter{exporter: exporter}
}

func (e *outputExporter) ExportOutput(key, value string) error {
	return e.exporter.ExportOutput(key, value)
}
