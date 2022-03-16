package meter

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

func GetMeter() metric.Meter {
	return global.Meter("k8sxds")
}
