package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type formatter struct{}

func newFormatter() *formatter {
	f := &formatter{}
	return f
}

func (f *formatter) format(entries []entry) string {
	out := ""
	for _, entry := range entries {
		if entry.isComment {
			out += entry.comment + "\n"
		} else {
			valStr := convFloat(entry.val)
			metricName := entry.labels.Get(nameLabel)
			labels := entry.labels.MatchLabels(false, nameLabel)
			labelsStr := labels.String()
			if labels.IsEmpty() {
				labelsStr = ""
			}
			format := fmt.Sprintf("%s%s %s", metricName, labelsStr, valStr)
			out += format + "\n"
		}
	}
	out = strings.TrimSuffix(out, "\n")
	return out
}

// common prom implementation
// https://github.com/prometheus/common/blob/7043ea0e691b6da9ecbd08e7ae41e9cf28898e98/expfmt/text_create.go#L432
func convFloat(f float64) string {
	switch {
	case f == 1:
		return "1"
	case f == 0:
		return "0"
	case f == -1:
		return "-1"
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, +1):
		return "+Inf"
	case math.IsInf(f, -1):
		return "-Inf"
	default:
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
}
