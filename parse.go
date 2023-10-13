package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/prometheus/prometheus/model/textparse"
)

const nameLabel = "__name__"

// entry is one line from a prom metrics file.
// Represents both comments and metrics.
type entry struct {
	val        float64
	ts         int64
	metricName string
	labels     labels.Labels
	lineNum    int
	isComment  bool
	comment    string
}

func newSeries(val float64, metricName string, labels labels.Labels, lineNum int) entry {
	e := entry{
		val:        val,
		metricName: metricName,
		labels:     labels,
		lineNum:    lineNum,
		isComment:  false,
	}
	return e
}

func newComment(text string, lineNum int, metricName string) entry {
	e := entry{
		metricName: metricName,
		lineNum:    lineNum,
		isComment:  true,
		comment:    text,
	}
	return e
}

// parseToSlice reads in lines and parses to entries.
// Relabelling is applied to each metric entry.
// Ensures each entry labelset is unique by combining
// values for duplicate labels.
func parseToSlice(buf []byte, relabelCfgs []*relabel.Config) ([]entry, error) {

	var err error
	var count int
	comments := make([]entry, 0)
	series := make(map[string]entry)
	parser := textparse.NewPromParser(buf)

	for {
		count++
		entry, err := parser.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				log.Print(err)
				break
			}
		}

		isHist := false
		isSeries := false
		switch entry {

		case textparse.EntryInvalid:
			continue

		case textparse.EntryHelp:
			metricName, help := parser.Help()
			text := fmt.Sprintf("# HELP %s %s", string(metricName), string(help))
			e := newComment(text, count, string(metricName))
			comments = append(comments, e)

		case textparse.EntryType:
			metricName, typ := parser.Type()
			text := fmt.Sprintf("# TYPE %s %s", string(metricName), string(typ))
			e := newComment(text, count, string(metricName))
			comments = append(comments, e)

		case textparse.EntryComment:
			comment := parser.Comment()
			text := fmt.Sprintf("%s", string(comment))
			e := newComment(text, count, "")
			comments = append(comments, e)

		case textparse.EntryHistogram:
			isHist = true
		case textparse.EntrySeries:
			isSeries = true
		default:
			continue
		}

		if isHist {
			var labels labels.Labels
			parser.Metric(&labels)
			metric, _, h, fh := parser.Histogram()
			fmt.Printf("\nlabels: %v", labels.String())
			fmt.Printf("\nmetric: %v", string(metric))
			if h != nil {
				fmt.Printf("\nh: %v", h)
			}
			if fh != nil {
				fmt.Printf("\nfh: %v", fh)
			}
			// TODO:
		}
		if isSeries {
			var labels labels.Labels
			parser.Metric(&labels)

			// Apply each relabel rule
			processedLabels, _ := relabel.Process(labels, relabelCfgs...)

			// TODO: support timestamps
			metric, _, val := parser.Series()

			labelsStr := processedLabels.String()

			if old, ok := series[labelsStr]; ok {
				old.val += val
				series[labelsStr] = old
			} else {
				e := newSeries(val, string(metric), processedLabels, count)
				series[labelsStr] = e
			}
		}
	}

	// Combine comments and series
	entries := comments
	for _, entry := range series {
		entries = append(entries, entry)
	}

	// Sort by line number
	// Need to ensure comments are next to their metircs.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].lineNum < entries[j].lineNum
	})

	return entries, err
}

func formatEntries(entries []entry) string {
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
