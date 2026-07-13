//go:build windows

package stdlib

import "yod/internal/object"

func winCreateChartBridge(args ...object.Object) object.Object {
	return winCreateChart(args...)
}

func newChartWidgetSafe(kind string) object.Object {
	return newChartWidget(kind)
}
