package metrichandlers

import "strings"

var (
	descReplacerFunc = strings.NewReplacer("_", " ", "-", " ", ".", " ")
	// TODO: re-check why we need below replacer, is it because of the replace char sequences
	metricReplacerFunc = strings.NewReplacer(".", "_", "-", "_", " ", "_")
)

// Utility functions

func NormalizeDesc(s string) string {
	return descReplacerFunc.Replace(s)
}

func NormalizeMetric(s string) string {
	return metricReplacerFunc.Replace(s)
}
