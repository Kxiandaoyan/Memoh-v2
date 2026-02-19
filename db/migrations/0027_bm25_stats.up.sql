-- 0027_bm25_stats
-- Persist BM25 corpus statistics (DocCount + AvgDocLen) across restarts
-- so the BM25 scoring is consistent without a full warm-up pass.

CREATE TABLE IF NOT EXISTS bm25_stats (
    lang        VARCHAR(32)              NOT NULL,
    doc_count   INTEGER                  NOT NULL DEFAULT 0,
    avg_doc_len DOUBLE PRECISION         NOT NULL DEFAULT 0,
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (lang)
);
