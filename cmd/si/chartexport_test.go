package main

import "testing"

func TestChartFile(t *testing.T) {
	for _, c := range []struct {
		n, k int
		ext  string
		want string
	}{
		{1, 0, "png", "talk-chart-1.png"},
		{2, 3, "svg", "talk-chart-2-step-3.svg"},
		{12, 1, "png", "talk-chart-12-step-1.png"},
	} {
		if got := chartFile("talk", c.n, c.k, c.ext); got != c.want {
			t.Errorf("chartFile(%d, %d, %s) = %s, want %s", c.n, c.k, c.ext, got, c.want)
		}
	}
	if !hasCharts([]byte(`<figure class="si-figure si-chart" aria-label="x">`)) || hasCharts([]byte(`<div class="si-ig-chart">`)) {
		t.Error("hasCharts: a chart fence is a si-chart figure, an infographic block is not")
	}
}
