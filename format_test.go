package proxy

import (
	"log/slog"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestFormat(t *testing.T) {
	p := NewParser(slog.Default())
	entries, err := p.parse([]byte(input1), []*relabel.Config{})
	if err != nil {
		t.Fatal(err)
	}

	f := NewFormatter(slog.Default())
	output := f.format(entries)

	if output != input1 {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(input1, output, false)
		t.Fatalf("input does not match output\ndiff:%s", dmp.DiffPrettyText(diffs))
	}
}

func TestSimpleRelabel(t *testing.T) {

	// Drop 'code' label from `http_requests_total`
	cfg := &relabel.Config{
		SourceLabels: model.LabelNames{"http_requests_total"},
		Regex:        relabel.MustNewRegexp("code"),
		Action:       relabel.LabelDrop,
	}

	p := NewParser(slog.Default())
	entries, err := p.parse([]byte(input2), []*relabel.Config{cfg})
	if err != nil {
		t.Fatal(err)
	}

	f := NewFormatter(slog.Default())
	output := f.format(entries)

	if output != output2 {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(input1, output, false)
		t.Fatalf("input does not match output\ndiff:%s", dmp.DiffPrettyText(diffs))
	}
}
