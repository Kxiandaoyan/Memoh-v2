package container

import (
	"github.com/Kxiandaoyan/Memoh-v2/internal/prune"
)

const (
	pruneHeadBytes = 4 * 1024
	pruneTailBytes = 1 * 1024
	pruneHeadLines = 150
	pruneTailLines = 50
)

func pruneToolOutputText(text, label string) string {
	return prune.PruneWithEdges(text, label, prune.Config{
		HeadBytes: pruneHeadBytes,
		TailBytes: pruneTailBytes,
		HeadLines: pruneHeadLines,
		TailLines: pruneTailLines,
	})
}
