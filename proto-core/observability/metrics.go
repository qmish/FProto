package observability

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	promexp "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// InitMeter sets up an OTel MeterProvider with Prometheus exporter.
func InitMeter() (*metric.MeterProvider, http.Handler, error) {
	exp, err := promexp.New()
	if err != nil {
		return nil, nil, fmt.Errorf("prometheus exporter: %w", err)
	}

	mp := metric.NewMeterProvider(metric.WithReader(exp))

	handler := promhttp.Handler()
	return mp, handler, nil
}
