---
name: x-tweet-fetcher
description: >
  Use when the user asks to fetch, read, or analyze tweets from X/Twitter (including replies, timelines, articles),
  or fetch content from Chinese platforms (Weibo, Bilibili, CSDN, WeChat).
  Also use for zero-cost Google search via Camofox.
  Basic tweet fetching requires only Python 3.7+. Replies/timelines/search require Camofox on localhost:9377.
---

# X Tweet Fetcher

Fetch tweets from X/Twitter without login or API keys. All scripts are in `$SKILLS_ROOT/x-tweet-fetcher/scripts/`.

## Prerequisites

```bash
python3 --version  # Requires Python 3.7+
```

## Commands

### Single Tweet (zero dependencies)

```bash
python3 $SKILLS_ROOT/x-tweet-fetcher/scripts/fetch_tweet.py --url "<TWEET_URL>" --pretty
```

Add `--text-only` for human-readable output.

### Reply Threads (requires Camofox on :9377)

```bash
python3 $SKILLS_ROOT/x-tweet-fetcher/scripts/fetch_tweet.py --url "<TWEET_URL>" --replies --pretty
```

### User Timeline (requires Camofox)

```bash
python3 $SKILLS_ROOT/x-tweet-fetcher/scripts/fetch_tweet.py --user <username> --limit 50 --pretty
```

### X Articles (long-form posts, requires Camofox)

```bash
python3 $SKILLS_ROOT/x-tweet-fetcher/scripts/fetch_tweet.py --article "<ARTICLE_URL_OR_ID>" --pretty
```

### Chinese Platforms (requires Camofox, except WeChat)

```bash
python3 $SKILLS_ROOT/x-tweet-fetcher/scripts/fetch_china.py --url "<URL>" --pretty
```

Supported: Weibo, Bilibili, CSDN, WeChat (WeChat works without Camofox).

### Google Search (requires Camofox)

```bash
python3 $SKILLS_ROOT/x-tweet-fetcher/scripts/camofox_client.py "search query"
```

## Output

All commands output JSON by default. Use `--text-only` for readable text, `--pretty` for formatted JSON.

## Camofox

Replies, timelines, articles, Chinese platforms, and Google search require Camofox running on `localhost:9377`. Check with:

```bash
curl -s http://localhost:9377/health
```
