package memory

import (
	"reflect"
	"testing"
)

func TestBM25Indexer_TermFrequencies(t *testing.T) {
	indexer := NewBM25Indexer(nil)

	tests := []struct {
		name    string
		lang    string
		text    string
		want    map[string]int
		docLen  int
		wantErr bool
	}{
		{
			name: "English text",
			lang: "en",
			text: "The quick brown fox jumps over the lazy dog",
			// Note: Bleve English analyzer stems words (jumps -> jump, lazy -> lazi) and removes stop words (the, over)
			want:   map[string]int{"quick": 1, "brown": 1, "fox": 1, "jump": 1, "lazi": 1, "dog": 1},
			docLen: 6,
		},
		{
			name: "CJK text",
			lang: "cjk",
			text: "你好世界",
			// Note: Bleve CJK analyzer uses bigrams
			want:   map[string]int{"你好": 1, "好世": 1, "世界": 1},
			docLen: 3,
		},
		{
			name: "Mixed text with standard analyzer",
			lang: "",
			text: "Go 语言 123",
			// Note: Standard analyzer splits CJK characters individually
			want:   map[string]int{"go": 1, "语": 1, "言": 1, "123": 1},
			docLen: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotLen, err := indexer.TermFrequencies(tt.lang, tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("TermFrequencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TermFrequencies() got = %v, want %v", got, tt.want)
			}
			if gotLen != tt.docLen {
				t.Errorf("TermFrequencies() gotLen = %v, want %v", gotLen, tt.docLen)
			}
		})
	}
}

func TestBM25Indexer_BM25Logic(t *testing.T) {
	indexer := NewBM25Indexer(nil)

	lang := "en"
	tf1 := map[string]int{"golang": 1, "programming": 1}
	len1 := 2
	indices1, values1 := indexer.AddDocument(lang, tf1, len1)

	tf2 := map[string]int{"golang": 1, "tutorial": 1, "advanced": 1, "topics": 1}
	len2 := 4
	indices2, values2 := indexer.AddDocument(lang, tf2, len2)

	// In BM25, same term in a shorter doc should have higher weight than in a longer doc.
	var weight1, weight2 float32
	for i, idx := range indices1 {
		if idx == termHash("golang") {
			weight1 = values1[i]
		}
	}
	for i, idx := range indices2 {
		if idx == termHash("golang") {
			weight2 = values2[i]
		}
	}

	if weight1 <= weight2 {
		t.Errorf("Expected weight in shorter doc (%f) to be higher than in longer doc (%f)", weight1, weight2)
	}

	// Add a doc without "golang" to increase doc count; IDF should increase.
	oldWeight1 := weight1
	indexer.AddDocument(lang, map[string]int{"rust": 1}, 1)
	indices3, values3 := indexer.AddDocument(lang, tf1, len1)

	for i, idx := range indices3 {
		if idx == termHash("golang") {
			weight1 = values3[i]
		}
	}

	if weight1 <= oldWeight1 {
		t.Errorf("Expected weight to increase as IDF increases (more docs without the term), got %f -> %f", oldWeight1, weight1)
	}
}

func TestBM25Indexer_RemoveDocument(t *testing.T) {
	indexer := NewBM25Indexer(nil)
	lang := "en"
	term := "test"

	tf, docLen, _ := indexer.TermFrequencies(lang, term)
	indexer.AddDocument(lang, tf, docLen)

	indexer.mu.RLock()
	stats := indexer.stats["en"]
	if stats.DocCount != 1 || stats.DocFreq[term] != 1 {
		t.Errorf("Expected stats to be updated after add, got count=%d, freq=%d", stats.DocCount, stats.DocFreq[term])
	}
	indexer.mu.RUnlock()

	indexer.RemoveDocument(lang, tf, docLen)

	indexer.mu.RLock()
	if stats.DocCount != 0 || stats.DocFreq[term] != 0 {
		t.Errorf("Expected stats to be cleared after remove, got count=%d, freq=%d", stats.DocCount, stats.DocFreq[term])
	}
	indexer.mu.RUnlock()
}

func TestTermHash_CollisionResistance(t *testing.T) {
	// Check that different terms get distinct hashes in 20-bit space (no collision in small sample).
	h1 := termHash("apple")
	h2 := termHash("orange")
	h3 := termHash("banana")

	if h1 == h2 || h2 == h3 || h1 == h3 {
		t.Errorf("Detected unexpected hash collision in small sample: %d, %d, %d", h1, h2, h3)
	}

	if h1 > sparseDimMask {
		t.Errorf("Hash %d exceeds mask %d", h1, sparseDimMask)
	}
}
