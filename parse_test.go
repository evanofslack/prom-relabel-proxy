package main

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	input1 = `# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{code="200", method="post"} 1.395066363e+12
http_requests_total{code="400", method="post"} 1.395066363e+12
http_requests_total{code="404", method="get"} 202021
http_requests_total{code="301", method="get"} 138291
# Escaping in label values:
msdos_file_access_time_seconds{error="Cannot find file:\n\"FILE.TXT\"", path="C:\\DIR\\FILE.TXT"} 1.458255915e+09
# Minimalistic line:
metric_without_timestamp_and_labels 12.47
# A weird metric from before the epoch:
something_weird{problem="division by zero"} 3.982045e+06
# A histogram, which has a pretty complex representation in the text format:
# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 24054
http_request_duration_seconds_bucket{le="0.1"} 33444
http_request_duration_seconds_bucket{le="0.2"} 100392
http_request_duration_seconds_bucket{le="0.5"} 129389
http_request_duration_seconds_bucket{le="1"} 133988
http_request_duration_seconds_bucket{le="+Inf"} 144320
http_request_duration_seconds_sum 53423
http_request_duration_seconds_count 144320
# Finally a summary, which has a complex representation, too:
# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
# TYPE rpc_duration_seconds summary
rpc_duration_seconds{quantile="0.01"} 3102
rpc_duration_seconds{quantile="0.05"} 3272
rpc_duration_seconds{quantile="0.5"} 4773
rpc_duration_seconds{quantile="0.9"} 9001
rpc_duration_seconds{quantile="0.99"} 76656
rpc_duration_seconds_sum 1.7560473e+07
rpc_duration_seconds_count 2693`

	input2 = `# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{code="200", method="post", hostname="node1"} 37821
http_requests_total{code="400", method="post", hostname="node1"} 992
http_requests_total{code="200", method="post", hostname="node2"} 48917
http_requests_total{code="400", method="post", hostname="node2"} 928
http_requests_total{code="200", method="get", hostname="node1"} 28920
http_requests_total{code="301", method="get", hostname="node1"} 802
http_requests_total{code="200", method="get", hostname="node2"} 81938
http_requests_total{code="301", method="get", hostname="node2"} 294`

	output2 = `# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{hostname="node1", method="post"} 38813
http_requests_total{hostname="node2", method="post"} 49845
http_requests_total{hostname="node1", method="get"} 29722
http_requests_total{hostname="node2", method="get"} 82232`
)

func TestParse(t *testing.T) {
	entries, err := parseToSlice([]byte(input1), []*relabel.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if want, got := 33, len(entries); want != got {
		t.Fatalf("Wrong entry length, wanted %d, got %d", want, got)
	}
}

func TestFormat(t *testing.T) {
	entries, err := parseToSlice([]byte(input1), []*relabel.Config{})
	if err != nil {
		t.Fatal(err)
	}

	output := formatEntries(entries)

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

	entries, err := parseToSlice([]byte(input2), []*relabel.Config{cfg})
	if err != nil {
		t.Fatal(err)
	}

	output := formatEntries(entries)

	if output != output2 {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(input1, output, false)
		t.Fatalf("input does not match output\ndiff:%s", dmp.DiffPrettyText(diffs))
	}
}
