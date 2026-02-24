#!/usr/bin/env python3
"""
China Platform Fetcher - Fetch posts from Chinese platforms.

Supported: Weibo, Bilibili, CSDN, WeChat (微信公众号).
Uses Camofox for server-side rendering, or direct HTTP for public pages.
Supports automatic platform detection and multiple output formats.
"""

import json
import re
import sys
import argparse
import time
import urllib.request
import urllib.error
from datetime import datetime, timezone, timedelta
from typing import Optional, Dict, List, Any
from abc import ABC, abstractmethod

# Import shared Camofox client
from camofox_client import (
    check_camofox,
    camofox_open_tab,
    camofox_snapshot,
    camofox_close_tab,
    camofox_fetch_page,
)


# ---------------------------------------------------------------------------
# i18n — bilingual messages (zh default, en via --lang en)
# ---------------------------------------------------------------------------

_MESSAGES = {
    "zh": {
        "opening_via_camofox": "[fetch_china] 正在通过 Camofox 打开 {url} ...",
        "camofox_not_running": (
            "Camofox 未在 localhost:{port} 运行。"
            "请先启动 Camofox。"
            "参考: https://github.com/openclaw/camofox"
        ),
        "snapshot_failed": "无法从 Camofox 获取页面快照",
        "platform_unsupported": "不支持的平台: {platform}",
        "url_not_supported": "URL 不匹配任何支持的平台",
        "parse_error": "解析页面内容失败",
        "comments_unavailable": "评论功能需要登录",
        "stats_views": "播放",
        "stats_likes": "点赞",
        "stats_comments": "评论",
        "stats_shares": "转发",
        "stats_bullets": "弹幕",
        "text_by": "作者: {author}",
        "text_at": "发布于: {time}",
        "text_stats": "❤️ {likes} | 💬 {comments} | 🔄 {shares} | 👀 {views}",
    },
    "en": {
        "opening_via_camofox": "[fetch_china] Opening {url} via Camofox...",
        "camofox_not_running": (
            "Camofox is not running on localhost:{port}. "
            "Please start Camofox first. "
            "See: https://github.com/openclaw/camofox"
        ),
        "snapshot_failed": "Failed to get page snapshot from Camofox",
        "platform_unsupported": "Unsupported platform: {platform}",
        "url_not_supported": "URL does not match any supported platform",
        "parse_error": "Failed to parse page content",
        "comments_unavailable": "Comments require login",
        "stats_views": "views",
        "stats_likes": "likes",
        "stats_comments": "comments",
        "stats_shares": "shares",
        "stats_bullets": "bullets",
        "text_by": "By: {author}",
        "text_at": "Posted at: {time}",
        "text_stats": "❤️ {likes} | 💬 {comments} | 🔄 {shares} | 👀 {views}",
    },
}

_lang: str = "zh"


def t(key: str, **kwargs) -> str:
    """Look up a message in the current language, formatting with kwargs."""
    msg = _MESSAGES.get(_lang, _MESSAGES["zh"]).get(key, key)
    return msg.format(**kwargs) if kwargs else msg


# ---------------------------------------------------------------------------
# Helper functions
# ---------------------------------------------------------------------------

def parse_wan_number(text: str) -> int:
    """Parse Chinese '万' number format (e.g., '77.7万' -> 777000, '1019.1万' -> 10191000)."""
    if not text:
        return 0
    text = text.strip()
    if "万" in text:
        try:
            num = float(text.replace("万", ""))
            return int(num * 10000)
        except ValueError:
            return 0
    # Try to parse as plain number
    try:
        return int(text.replace(",", ""))
    except ValueError:
        return 0


# ---------------------------------------------------------------------------
# Platform patterns and identification
# ---------------------------------------------------------------------------

PLATFORM_PATTERNS = {
    'weibo': r'weibo\.(com|cn)',
    'bilibili': r'bilibili\.com|b23\.tv',
    'csdn': r'blog\.csdn\.net|csdn\.net',
    'weixin': r'mp\.weixin\.qq\.com',
    'douyin': r'douyin\.com|v\.douyin\.com',
}


def identify_platform(url: str) -> Optional[str]:
    """Auto-detect platform from URL."""
    for platform, pattern in PLATFORM_PATTERNS.items():
        if re.search(pattern, url, re.IGNORECASE):
            return platform
    return None


# ---------------------------------------------------------------------------
# Base parser class
# ---------------------------------------------------------------------------

class PlatformParser(ABC):
    """Abstract base class for platform-specific parsers."""

    name: str = "unknown"

    @abstractmethod
    def can_handle(self, url: str) -> bool:
        """Check if this parser can handle the given URL."""
        pass

    @abstractmethod
    def fetch(self, url: str, port: int = 9377) -> Dict[str, Any]:
        """Fetch and parse content from the URL."""
        pass

    @abstractmethod
    def to_markdown(self, data: Dict[str, Any]) -> str:
        """Convert parsed data to Markdown format."""
        pass

    def to_text(self, data: Dict[str, Any]) -> str:
        """Convert parsed data to human-readable text."""
        lines = []
        if data.get("title"):
            lines.append(f"# {data['title']}\n")
        if data.get("author"):
            lines.append(t("text_by", author=data["author"]))
        if data.get("published_at"):
            lines.append(t("text_at", time=data["published_at"]))

        stats = data.get("stats", {})
        if stats:
            lines.append(t(
                "text_stats",
                likes=stats.get("likes", 0),
                comments=stats.get("comments", 0),
                shares=stats.get("shares", 0),
                views=stats.get("views", 0),
            ))

        if data.get("content"):
            lines.append(f"\n{data['content']}")

        comments = data.get("comments", [])
        if comments:
            lines.append(f"\n## Comments ({len(comments)})\n")
            for c in comments[:10]:
                lines.append(f"- **{c.get('author', '')}**: {c.get('text', '')}")

        return "\n".join(lines)


# ---------------------------------------------------------------------------
# Weibo parser
# ---------------------------------------------------------------------------

class WeiboParser(PlatformParser):
    """Parser for Weibo posts."""

    name = "weibo"

    def can_handle(self, url: str) -> bool:
        return bool(re.search(r'weibo\.(com|cn)', url, re.IGNORECASE))

    def fetch(self, url: str, port: int = 9377) -> Dict[str, Any]:
        if not check_camofox(port):
            return {"url": url, "platform": "weibo", "error": t("camofox_not_running", port=port)}

        print(t("opening_via_camofox", url=url), file=sys.stderr)

        session_key = f"weibo-{int(time.time())}"
        snapshot = camofox_fetch_page(url, session_key, wait=8, port=port)

        if not snapshot:
            return {"url": url, "platform": "weibo", "error": t("snapshot_failed")}

        # Parse the snapshot
        data = self._parse_snapshot(snapshot, url)
        return data

    def _parse_snapshot(self, snapshot: str, url: str) -> Dict[str, Any]:
        """Parse Weibo page snapshot.
        
        Real snapshot format (each post is an article: block):
        - article:
            - link "作者名" [eN]:
                - /url: //weibo.com/u/UID
            - link "2小时前" [eN]:
                - /url: https://weibo.com/UID/PostID
            - text: 身份描述  (认证信息)
            - text: 正文内容...  (帖子正文，可能多行)
            - link "#话题#" [eN]:
            - text:  241  102  (转发 评论)
            - button "1793" [eN]:  (点赞数)
        
        Returns the first article's data.
        """
        lines = snapshot.split("\n")

        # Find all article blocks - simplified
        articles = []
        current_article = None
        in_article = False

        for i, line in enumerate(lines):
            stripped = line.rstrip()
            
            # Start of article (may have leading spaces)
            if stripped.lstrip().startswith("- article:"):
                current_article = {
                    "author": "",
                    "author_url": "",
                    "time": "",
                    "time_url": "",
                    "verified_text": "",
                    "content": "",
                    "topics": [],
                    "shares": 0,
                    "comments": 0,
                    "likes": 0,
                }
                in_article = True
                continue

            if not in_article or current_article is None:
                continue

            # End of article - next article starts or we see non-article content
            stripped_lstrip = stripped.lstrip()
            if stripped_lstrip.startswith("- article:"):
                articles.append(current_article)
                current_article = {
                    "author": "", "author_url": "", "time": "", "time_url": "",
                    "verified_text": "", "content": "", "topics": [],
                    "shares": 0, "comments": 0, "likes": 0,
                }
                continue

            # Only process content inside article - elements start with "    - " (4 spaces + dash)
            if not stripped.startswith("    - "):
                continue

            # Author: "    - link \"Name\" [eN]:" with URL containing weibo.com/u/
            if not current_article["author"] and stripped.startswith("    - link "):
                # Check next line for URL
                if i + 1 < len(lines):
                    next_line = lines[i + 1].rstrip()
                    if "/url:" in next_line and "weibo.com/u/" in next_line:
                        match = re.match(r'^- link "([^"]+)"', stripped.lstrip())
                        if match:
                            current_article["author"] = match.group(1)
                            # Extract URL
                            url_match = re.search(r'/url:\s*(//?[^$]+)', next_line)
                            if url_match:
                                current_article["author_url"] = url_match.group(1).strip()
                        continue

            # Time: "    - link \"X小时前\" [eN]:" with post URL
            if not current_article["time"] and stripped.startswith("    - link "):
                if i + 1 < len(lines):
                    next_line = lines[i + 1].rstrip()
                    if "/url:" in next_line and "weibo.com/" in next_line:
                        # Make sure this isn't the author link
                        if "weibo.com/u/" not in stripped:
                            match = re.match(r'^- link "([^"]+)"', stripped.lstrip())
                            if match:
                                time_text = match.group(1)
                                # Verify it's a time pattern
                                if re.search(r'^\d+[时分秒]前$', time_text) or re.search(r'^\d+-\d+\s+\d+:\d+$', time_text):
                                    current_article["time"] = time_text
                                    # Extract post URL
                                    url_match = re.search(r'/url:\s*(https?://[^$]+)', next_line)
                                    if url_match:
                                        current_article["time_url"] = url_match.group(1).strip()
                            continue

            # Verified text (认证信息): "    - text: 认证信息"
            if current_article["author"] and not current_article["verified_text"]:
                if stripped.startswith("    - text:"):
                    text = stripped[len("    - text:"):].strip()
                    # Skip common UI text
                    if text and text.lower() not in ("转发", "评论", "赞", "收藏", "更多", ""):
                        # Check if it's verified info (contains "已编辑" or short)
                        if "已编辑" in text or len(text) < 60:
                            current_article["verified_text"] = text
                        elif not current_article["content"]:
                            current_article["content"] = text
                        continue

            # Topics: "    - link \"#话题#\""
            if stripped.startswith("    - link #"):
                match = re.match(r'^- link "(#[^#]+#)"', stripped.lstrip())
                if match:
                    current_article["topics"].append(match.group(1))

            # Content: "    - text: 正文" (after verified text)
            if current_article.get("verified_text") and not current_article.get("content"):
                if stripped.startswith("    - text:"):
                    text = stripped[len("    - text:"):].strip()
                    if text and len(text) > 5:
                        current_article["content"] = text
                        continue

            # Stats: "    - text:  241  102" (转发 评论)
            if stripped.startswith("    - text:") and "转发" not in stripped:
                text = stripped[len("    - text:"):].strip()
                # Match pattern with numbers (possibly with emoji icons)
                nums = re.findall(r'(\d+(?:\.\d+)?万?)', text)
                if len(nums) >= 2:
                    try:
                        current_article["shares"] = parse_wan_number(nums[0])
                        current_article["comments"] = parse_wan_number(nums[1])
                    except:
                        pass

            # Likes: "    - button \"1793\" [eN]:"
            if stripped.startswith("    - button "):
                match = re.match(r'^- button "(\d+(?:\.\d+)?万?)"', stripped.lstrip())
                if match:
                    current_article["likes"] = parse_wan_number(match.group(1))

        # Add last article if still open
        if in_article and current_article is not None:
            articles.append(current_article)

        # Use first article
        if not articles:
            return {
                "url": url,
                "platform": "weibo",
                "title": "",
                "author": "未知",
                "published_at": "",
                "fetched_at": datetime.now(timezone(timedelta(hours=8))).isoformat(),
                "content": "",
                "stats": {"likes": 0, "comments": 0, "shares": 0, "views": 0},
                "media": [],
                "comments": [],
                "availability": "partial",
                "unavailable_fields": ["comments"],
            }

        first = articles[0]
        
        # Build content: verified text + main content
        full_content = ""
        if first.get("verified_text"):
            full_content = first["verified_text"]
        if first.get("content"):
            if full_content:
                full_content += "\n" + first["content"]
            else:
                full_content = first["content"]

        result = {
            "url": first.get("time_url") or url,
            "platform": "weibo",
            "title": "",
            "author": first.get("author", "未知"),
            "author_handle": first.get("author_url", ""),
            "published_at": first.get("time", ""),
            "fetched_at": datetime.now(timezone(timedelta(hours=8))).isoformat(),
            "content": full_content,
            "stats": {
                "likes": first.get("likes", 0),
                "comments": first.get("comments", 0),
                "shares": first.get("shares", 0),
                "views": 0,
            },
            "media": [],
            "topics": first.get("topics", []),
            "comments": [],
            "availability": "partial",
            "unavailable_fields": ["comments"],
        }

        return result

    def to_markdown(self, data: Dict[str, Any]) -> str:
        lines = [
            "---",
            f"platform: {data['platform']}",
            f"url: {data['url']}",
            f"title: \"{data.get('title', '')}\"",
            f"author: \"{data.get('author', '')}\"",
            f"published_at: \"{data.get('published_at', '')}\"",
            f"fetched_at: \"{data.get('fetched_at', '')}\"",
            "stats:",
            f"  likes: {data.get('stats', {}).get('likes', 0)}",
            f"  comments: {data.get('stats', {}).get('comments', 0)}",
            f"  shares: {data.get('stats', {}).get('shares', 0)}",
            f"availability: {data.get('availability', 'full')}",
            "---",
            "",
        ]

        if data.get("title"):
            lines.append(f"# {data['title']}\n")

        if data.get("content"):
            lines.append(data["content"])

        return "\n".join(lines)


# ---------------------------------------------------------------------------
# Bilibili parser
# ---------------------------------------------------------------------------

class BilibiliParser(PlatformParser):
    """Parser for Bilibili videos."""

    name = "bilibili"

    def can_handle(self, url: str) -> bool:
        return bool(re.search(r'bilibili\.com|b23\.tv', url, re.IGNORECASE))

    def fetch(self, url: str, port: int = 9377) -> Dict[str, Any]:
        if not check_camofox(port):
            return {"url": url, "platform": "bilibili", "error": t("camofox_not_running", port=port)}

        print(t("opening_via_camofox", url=url), file=sys.stderr)

        session_key = f"bilibili-{int(time.time())}"
        snapshot = camofox_fetch_page(url, session_key, wait=8, port=port)

        if not snapshot:
            return {"url": url, "platform": "bilibili", "error": t("snapshot_failed")}

        data = self._parse_snapshot(snapshot, url)
        return data

    def _parse_snapshot(self, snapshot: str, url: str) -> Dict[str, Any]:
        """Parse Bilibili video page snapshot.
        
        Real snapshot format:
        - heading "标题" [level=1]
        - text: 1019.1万  (播放量)
        - text: 1.1万 2026-02-17 23:51:30  (弹幕+时间)
        - text: 未经作者授权...  (简介)
        - text: 77.7万  (点赞)
        - text: 8.8万  (投币)
        - text: 19.8万  (收藏)
        - text: 19.1万  (转发)
        - link "UP主名" [eN]:  (UP主，URL包含space.bilibili.com)
        - text: 关注 61.8万  (粉丝数)
        """
        lines = snapshot.split("\n")

        title = ""
        author = ""
        description = ""
        published_at = ""
        views = 0
        bullets = 0
        likes = 0
        coins = 0
        favorites = 0
        shares = 0
        followers = 0

        heading_found = False
        stats_started = False
        stats_count = 0
        
        i = 0
        while i < len(lines):
            line = lines[i]
            stripped = line.strip()

            # 1. 标题: heading "..." [level=1]
            if not title and stripped.startswith('- heading "'):
                match = re.match(r'^- heading "(.+)" \[level=(\d+)\]', stripped)
                if match:
                    title = match.group(1)
                    heading_found = True
                    i += 1
                    continue

            # 如果还没找到 heading，继续下一行
            if not heading_found:
                i += 1
                continue

            # 2. 播放量: heading后第一个包含"万"的text行（不包含日期）
            if heading_found and not views and stripped.startswith("- text:"):
                text_content = stripped[len("- text:"):].strip()
                # 播放量：纯数字+万，不包含日期时间
                if "万" in text_content and not re.search(r'\d{4}-\d{2}-\d{2}', text_content):
                    # 可能是播放量 (1019.1万)
                    # 排除简介（未经作者授权）
                    if not text_content.startswith("未经"):
                        views = parse_wan_number(text_content)
                        i += 1
                        continue

            # 3. 弹幕+发布时间: 包含日期格式 YYYY-MM-DD HH:MM:SS 的行
            if not published_at and stripped.startswith("- text:"):
                text_content = stripped[len("- text:"):].strip()
                # 匹配日期时间格式
                date_match = re.search(r'(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})', text_content)
                if date_match:
                    published_at = date_match.group(1)
                    # 尝试提取弹幕数 (1.1万 2026-02-17 23:51:30)
                    bullet_match = re.search(r'([\d.]+万)', text_content)
                    if bullet_match:
                        bullets = parse_wan_number(bullet_match.group(1))
                    i += 1
                    continue

            # 4. 简介: text: 未经作者授权 开头的行
            if not description and stripped.startswith("- text:"):
                text_content = stripped[len("- text:"):].strip()
                if text_content.startswith("未经作者授权") or text_content.startswith("未经"):
                    description = text_content
                    i += 1
                    continue

            # 5. 互动数据: 在"发送"按钮后连续出现4个带"万"的text行 (点赞、投币、收藏、转发)
            # 格式: - text: 77.7万 \n - img \n - text: 8.8万 \n - img ...
            # 所以我们需要找连续4个包含"万"且格式为 X.X万 的 text 行
            if not stats_started and stripped.startswith("- text:"):
                text_content = stripped[len("- text:"):].strip()
                # 检查是否是互动数据格式 (如 77.7万)
                if re.match(r'^[\d.]+万$', text_content):
                    # 这可能是第一个互动数据
                    # 检查前后是否有img
                    stats_started = True
                    stats_count = 1
                    likes = parse_wan_number(text_content)
                    
                    # 继续检查接下来的行
                    j = i + 1
                    while j < len(lines) and stats_count < 4:
                        next_stripped = lines[j].strip()
                        if next_stripped.startswith("- text:"):
                            next_text = next_stripped[len("- text:"):].strip()
                            if re.match(r'^[\d.]+万$', next_text):
                                stats_count += 1
                                if stats_count == 2:
                                    coins = parse_wan_number(next_text)
                                elif stats_count == 3:
                                    favorites = parse_wan_number(next_text)
                                elif stats_count == 4:
                                    shares = parse_wan_number(next_text)
                            else:
                                break
                        elif next_stripped == "- img":
                            # img 行，继续
                            pass
                        else:
                            break
                        j += 1
                    
                    # 跳过已检查的行
                    i = j
                    continue
            
            # 6. UP主: link "UP主名" [eN]: 且URL包含 space.bilibili.com
            if not author and stripped.startswith("- link "):
                # 检查下一行是否是 space.bilibili.com URL
                if i + 1 < len(lines):
                    next_line = lines[i + 1].strip()
                    if "/url:" in next_line and "space.bilibili.com" in next_line:
                        match = re.match(r'^- link "([^"]+)"\s*(\[e\d+\])?:?$', stripped)
                        if match:
                            author = match.group(1)
                            # 跳过检查 next_line
                            i += 2
                            continue

            # 7. 粉丝数: text: 关注 61.8万
            if not followers and stripped.startswith("- text:"):
                text_content = stripped[len("- text:"):].strip()
                # 匹配 "关注 数字万"
                follow_match = re.match(r'^关注\s+([\d.]+万)$', text_content)
                if follow_match:
                    followers = parse_wan_number(follow_match.group(1))

            i += 1

        result = {
            "url": url,
            "platform": "bilibili",
            "title": title,
            "author": author or "未知UP主",
            "published_at": published_at,
            "fetched_at": datetime.now(timezone(timedelta(hours=8))).isoformat(),
            "content": description,
            "stats": {
                "likes": likes,
                "comments": 0,  # Requires login
                "shares": shares,
                "views": views,
                "bullets": bullets,
                "coins": coins,
                "favorites": favorites,
                "followers": followers,
            },
            "media": [],
            "comments": [],
            "availability": "partial",
            "unavailable_fields": ["comments"],
        }

        return result

    def to_markdown(self, data: Dict[str, Any]) -> str:
        lines = [
            "---",
            f"platform: {data['platform']}",
            f"url: {data['url']}",
            f"title: \"{data.get('title', '')}\"",
            f"author: \"{data.get('author', '')}\"",
            f"published_at: \"{data.get('published_at', '')}\"",
            f"fetched_at: \"{data.get('fetched_at', '')}\"",
            "stats:",
            f"  likes: {data.get('stats', {}).get('likes', 0)}",
            f"  views: {data.get('stats', {}).get('views', 0)}",
            f"  comments: {data.get('stats', {}).get('comments', 0)}",
            f"  shares: {data.get('stats', {}).get('shares', 0)}",
            f"  bullets: {data.get('stats', {}).get('bullets', 0)}",
            f"availability: {data.get('availability', 'partial')}",
            "---",
            "",
        ]

        if data.get("title"):
            lines.append(f"# {data['title']}\n")

        if data.get("tags"):
            lines.append(f"**标签**: {' '.join('#' + tag for tag in data['tags'])}\n")

        if data.get("content"):
            lines.append(f"## 简介\n{data['content']}")

        return "\n".join(lines)


# ---------------------------------------------------------------------------
# CSDN parser
# ---------------------------------------------------------------------------

class CSDNParser(PlatformParser):
    """Parser for CSDN blog articles."""

    name = "csdn"

    def can_handle(self, url: str) -> bool:
        return bool(re.search(r'blog\.csdn\.net|csdn\.net', url, re.IGNORECASE))

    def fetch(self, url: str, port: int = 9377) -> Dict[str, Any]:
        if not check_camofox(port):
            return {"url": url, "platform": "csdn", "error": t("camofox_not_running", port=port)}

        print(t("opening_via_camofox", url=url), file=sys.stderr)

        session_key = f"csdn-{int(time.time())}"
        snapshot = camofox_fetch_page(url, session_key, wait=8, port=port)

        if not snapshot:
            return {"url": url, "platform": "csdn", "error": t("snapshot_failed")}

        data = self._parse_snapshot(snapshot, url)
        return data

    def _parse_snapshot(self, snapshot: str, url: str) -> Dict[str, Any]:
        """Parse CSDN page snapshot.
        
        Real snapshot format for download list page:
        - listitem with link containing file info and URL
          e.g., "1.69MB 强化学习算法在大语言模型...zip 2026-02-19"
        
        For article pages, typical format:
        - heading "文章标题" [level=1]
        - link "作者名" [eN]:
        - text: 发布时间
        - text: 阅读数, 点赞数, 评论数
        - text: 文章内容...
        """
        lines = snapshot.split("\n")

        title = ""
        author = ""
        published_at = ""
        content = ""
        views = 0
        likes = 0
        comments_count = 0
        
        # Try to detect page type
        is_download_page = False
        downloads = []
        
        # Check if it's a download list (contains file sizes like "1.69MB", "201KB")
        if "MB" in snapshot or "KB" in snapshot:
            is_download_page = True

        if is_download_page:
            # Parse as download list
            for i, line in enumerate(lines):
                stripped = line.strip()
                
                # Download items: link with file info
                if stripped.startswith("- listitem:"):
                    # Check next few lines for link
                    for j in range(1, 5):
                        if i + j < len(lines):
                            next_line = lines[i + j].strip()
                            if next_line.startswith("- link "):
                                # Check for URL
                                if i + j + 1 < len(lines):
                                    url_line = lines[i + j + 1].strip()
                                    if "/url:" in url_line:
                                        # Extract file info from link text
                                        match = re.match(r'^- link "([^"]+)"\s*(\[e\d+\])?:?$', next_line)
                                        if match:
                                            link_text = match.group(1)
                                            # Extract file size, name, date
                                            # Pattern: "1.69MB 文件名 2026-02-19"
                                            file_match = re.match(r'^([\d.]+(?:MB|KB))\s+(.+?)\s+(\d{4}-\d{2}-\d{2})$', link_text)
                                            if file_match:
                                                size = file_match.group(1)
                                                filename = file_match.group(2)
                                                date = file_match.group(3)
                                                
                                                # Extract URL
                                                url_match = re.search(r'/url:\s*(https?://[^$]+)', url_line)
                                                file_url = url_match.group(1).strip() if url_match else ""
                                                
                                                downloads.append({
                                                    "filename": filename,
                                                    "size": size,
                                                    "date": date,
                                                    "url": file_url,
                                                })
                                        break
        
        # If not download page, try to parse as article
        if not is_download_page:
            heading_found = False
            for i, line in enumerate(lines):
                stripped = line.strip()

                # 1. Title: heading level 1
                if not title and stripped.startswith('- heading "'):
                    match = re.match(r'^- heading "(.+)" \[level=(\d+)\]', stripped)
                    if match:
                        title = match.group(1)
                        heading_found = True
                        continue

                # 2. Author
                if not author and stripped.startswith("- link "):
                    # Check if next line has profile URL
                    if i + 1 < len(lines):
                        next_line = lines[i + 1].strip()
                        if "/url:" in next_line:
                            match = re.match(r'^- link "([^"]+)"\s*(\[e\d+\])?:?$', stripped)
                            if match:
                                author = match.group(1)
                                continue

                # 3. Published time
                if not published_at and stripped.startswith("- text:"):
                    text = stripped[len("- text:"):].strip()
                    # Match date pattern
                    date_match = re.match(r'^(\d{4}-\d{2}-\d{2})$', text)
                    if date_match:
                        published_at = date_match.group(1)
                        continue

                # 4. Stats (阅读, 点赞, 评论)
                if stripped.startswith("- text:"):
                    text = stripped[len("- text:"):].strip()
                    # Match patterns like "1000阅读" or "1000 阅读"
                    views_match = re.search(r'([\d,]+)\s*阅读', text)
                    if views_match and views == 0:
                        views = int(views_match.group(1).replace(",", ""))
                    
                    likes_match = re.search(r'([\d,]+)\s*点赞', text)
                    if likes_match and likes == 0:
                        likes = int(likes_match.group(1).replace(",", ""))
                    
                    comments_match = re.search(r'([\d,]+)\s*评论', text)
                    if comments_match and comments_count == 0:
                        comments_count = int(comments_match.group(1).replace(",", ""))

                # 5. Content
                if heading_found and stripped.startswith("- text:") and len(stripped) > 20:
                    text = stripped[len("- text:"):].strip()
                    # Skip UI elements
                    if text and text.lower() not in ("编辑", "删除", "收藏", "举报", "分享", "返回", "评论"):
                        content += "\n" + text

        result = {
            "url": url,
            "platform": "csdn",
            "title": title,
            "author": author or "未知作者",
            "published_at": published_at,
            "fetched_at": datetime.now(timezone(timedelta(hours=8))).isoformat(),
            "content": content if not is_download_page else f"共 {len(downloads)} 个下载资源",
            "stats": {
                "likes": likes,
                "comments": comments_count,
                "shares": 0,
                "views": views,
            },
            "media": [],
            "downloads": downloads if is_download_page else [],
            "comments": [],
            "availability": "partial" if not comments_count else "full",
            "unavailable_fields": ["comments"] if not comments_count else [],
        }

        return result

    def to_markdown(self, data: Dict[str, Any]) -> str:
        lines = [
            "---",
            f"platform: {data['platform']}",
            f"url: {data['url']}",
            f"title: \"{data.get('title', '')}\"",
            f"author: \"{data.get('author', '')}\"",
            f"published_at: \"{data.get('published_at', '')}\"",
            f"fetched_at: \"{data.get('fetched_at', '')}\"",
            "stats:",
            f"  likes: {data.get('stats', {}).get('likes', 0)}",
            f"  views: {data.get('stats', {}).get('views', 0)}",
            f"  comments: {data.get('stats', {}).get('comments', 0)}",
            f"availability: {data.get('availability', 'partial')}",
            "---",
            "",
        ]

        if data.get("title"):
            lines.append(f"# {data['title']}\n")

        if data.get("toc"):
            lines.append("## 目录\n")
            for item in data["toc"]:
                lines.append(f"- {item}")
            lines.append("")

        if data.get("content"):
            lines.append(data["content"])

        return "\n".join(lines)


# ---------------------------------------------------------------------------
# WeChat (微信公众号) parser — no Camofox needed, direct HTTP
# ---------------------------------------------------------------------------

class WeixinParser(PlatformParser):
    """Parser for WeChat Official Account articles (mp.weixin.qq.com)."""

    name = "weixin"

    def can_handle(self, url: str) -> bool:
        return bool(re.search(r'mp\.weixin\.qq\.com', url, re.IGNORECASE))

    def fetch(self, url: str, port: int = 9377) -> Dict[str, Any]:
        """Fetch WeChat article via direct HTTP (public pages, no login needed)."""
        print(f"[fetch_china] 正在抓取微信公众号文章 {url} ...", file=sys.stderr)

        try:
            req = urllib.request.Request(url, headers={
                "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
            })
            with urllib.request.urlopen(req, timeout=15) as resp:
                html = resp.read().decode("utf-8", errors="replace")
        except Exception as e:
            # Fallback to Camofox if direct HTTP fails
            if check_camofox(port):
                print("[fetch_china] HTTP 失败，尝试 Camofox ...", file=sys.stderr)
                snapshot = camofox_fetch_page(url, f"weixin-{int(time.time())}", wait=8, port=port)
                if snapshot:
                    return self._parse_snapshot(snapshot, url)
            return {"url": url, "platform": "weixin", "error": f"抓取失败: {e}"}

        return self._parse_html(html, url)

    def _parse_html(self, html: str, url: str) -> Dict[str, Any]:
        """Parse WeChat article from raw HTML."""
        title = ""
        author = ""
        account = ""
        published_at = ""
        content = ""

        # Title: <meta property="og:title" content="...">
        m = re.search(r'<meta\s+property="og:title"\s+content="([^"]*)"', html)
        if m:
            title = self._unescape_html(m.group(1))
        if not title:
            m = re.search(r'<h1[^>]*class="rich_media_title"[^>]*>(.*?)</h1>', html, re.DOTALL)
            if m:
                title = re.sub(r'<[^>]+>', '', m.group(1)).strip()

        # Author: <meta name="author" content="...">
        m = re.search(r'<meta\s+name="author"\s+content="([^"]*)"', html)
        if m:
            author = self._unescape_html(m.group(1))

        # Account name: var nickname = "..." or <a id="js_name">...</a>
        m = re.search(r'var\s+nickname\s*=\s*["\']([^"\']+)["\']', html)
        if m:
            account = m.group(1)
        if not account:
            m = re.search(r'<a[^>]*id="js_name"[^>]*>(.*?)</a>', html, re.DOTALL)
            if m:
                account = re.sub(r'<[^>]+>', '', m.group(1)).strip()

        # Published time: var ct = "timestamp"
        m = re.search(r'var\s+ct\s*=\s*["\'](\d+)["\']', html)
        if m:
            ts = int(m.group(1))
            dt = datetime.fromtimestamp(ts, tz=timezone(timedelta(hours=8)))
            published_at = dt.strftime("%Y-%m-%d %H:%M:%S")

        # Content: <div class="rich_media_content" ...>...</div>
        m = re.search(
            r'<div[^>]*class="rich_media_content[^"]*"[^>]*>(.*?)</div>\s*(?:<div|<script)',
            html, re.DOTALL
        )
        if m:
            raw = m.group(1)
            # Strip HTML tags but preserve paragraphs
            raw = re.sub(r'<br\s*/?>', '\n', raw)
            raw = re.sub(r'</p>', '\n', raw)
            raw = re.sub(r'<[^>]+>', '', raw)
            # Clean up whitespace
            raw = re.sub(r'&nbsp;', ' ', raw)
            raw = self._unescape_html(raw)
            lines = [line.strip() for line in raw.split('\n') if line.strip()]
            content = '\n'.join(lines)

        # Extract images from og:image or content
        images = []
        for img_match in re.finditer(r'data-src="(https?://mmbiz[^"]+)"', html):
            img_url = img_match.group(1)
            if img_url not in images:
                images.append(img_url)

        result = {
            "url": url,
            "platform": "weixin",
            "title": title,
            "author": author or account or "未知公众号",
            "account": account,
            "published_at": published_at,
            "fetched_at": datetime.now(timezone(timedelta(hours=8))).isoformat(),
            "content": content,
            "stats": {
                "likes": 0,
                "comments": 0,
                "shares": 0,
                "views": 0,
            },
            "media": images,
            "comments": [],
            "availability": "full" if content else "partial",
            "unavailable_fields": ["stats"],
        }
        return result

    def _parse_snapshot(self, snapshot: str, url: str) -> Dict[str, Any]:
        """Fallback: parse from Camofox snapshot."""
        lines = snapshot.split("\n")
        title = ""
        content_parts = []

        for line in lines:
            stripped = line.strip()
            if not title and stripped.startswith("- heading "):
                m = re.match(r'^- heading "(.+?)"\s*\[level=\d\]', stripped)
                if m:
                    title = m.group(1)
            if stripped.startswith("- text:"):
                text = stripped[len("- text:"):].strip()
                if text and len(text) > 5:
                    content_parts.append(text)

        return {
            "url": url,
            "platform": "weixin",
            "title": title,
            "author": "未知公众号",
            "published_at": "",
            "fetched_at": datetime.now(timezone(timedelta(hours=8))).isoformat(),
            "content": "\n".join(content_parts),
            "stats": {"likes": 0, "comments": 0, "shares": 0, "views": 0},
            "media": [],
            "comments": [],
            "availability": "partial",
            "unavailable_fields": ["stats", "author"],
        }

    @staticmethod
    def _unescape_html(text: str) -> str:
        """Unescape common HTML entities."""
        text = text.replace("&amp;", "&")
        text = text.replace("&lt;", "<")
        text = text.replace("&gt;", ">")
        text = text.replace("&quot;", '"')
        text = text.replace("&#39;", "'")
        text = text.replace("&nbsp;", " ")
        return text

    def to_markdown(self, data: Dict[str, Any]) -> str:
        lines = [
            "---",
            f"platform: {data['platform']}",
            f"url: {data['url']}",
            f"title: \"{data.get('title', '')}\"",
            f"author: \"{data.get('author', '')}\"",
            f"account: \"{data.get('account', '')}\"",
            f"published_at: \"{data.get('published_at', '')}\"",
            f"fetched_at: \"{data.get('fetched_at', '')}\"",
            f"availability: {data.get('availability', 'full')}",
            "---",
            "",
        ]
        if data.get("title"):
            lines.append(f"# {data['title']}\n")
        if data.get("content"):
            lines.append(data["content"])
        if data.get("media"):
            lines.append("\n\n## 图片\n")
            for i, img in enumerate(data["media"][:10], 1):
                lines.append(f"![图片{i}]({img})")
        return "\n".join(lines)


# ---------------------------------------------------------------------------
# Douyin (抖音)
# ---------------------------------------------------------------------------

class DouyinParser(PlatformParser):
    """Parser for Douyin (抖音) videos — extracts AI chapter summaries."""

    name = "douyin"

    def can_handle(self, url: str) -> bool:
        return bool(re.search(r'douyin\.com|v\.douyin\.com', url, re.IGNORECASE))

    def _resolve_short_url(self, url: str) -> str:
        """Resolve v.douyin.com short URLs to full douyin.com URLs."""
        if 'v.douyin.com' not in url:
            return url
        try:
            req = urllib.request.Request(url, method='HEAD')
            req.add_header('User-Agent', 'Mozilla/5.0')
            resp = urllib.request.urlopen(req, timeout=10)
            return resp.url
        except Exception:
            return url

    def fetch(self, url: str, port: int = 9377) -> Dict[str, Any]:
        if not check_camofox(port):
            return {"url": url, "platform": "douyin", "error": t("camofox_not_running", port=port)}

        print(t("opening_via_camofox", url=url), file=sys.stderr)

        # Resolve short URL
        resolved = self._resolve_short_url(url)

        session_key = f"douyin-{int(time.time())}"
        snapshot = camofox_fetch_page(resolved, session_key, wait=12, port=port)

        if not snapshot:
            return {"url": url, "platform": "douyin", "error": t("snapshot_failed")}

        data = self._parse_snapshot(snapshot, url)
        return data

    def _parse_snapshot(self, snapshot: str, url: str) -> Dict[str, Any]:
        """Parse Douyin video page snapshot."""
        lines = snapshot.split("\n")

        title = ""
        author = ""
        description = ""
        published_at = ""
        likes = 0
        comments = 0
        favorites = 0
        shares = 0
        chapters = []

        i = 0
        while i < len(lines):
            line = lines[i].strip()

            # Title from heading
            m = re.search(r'heading "(.+?)"', line)
            if m and not title:
                title = m.group(1)

            # Author — typically a link to user profile
            if 'douyin.com/user/' in line:
                m2 = re.search(r'link "(.+?)"', line)
                if m2 and not author:
                    author = m2.group(1)

            # Published time — e.g. "2026-02-20 06:19"
            m_time = re.search(r'(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2})', line)
            if m_time and not published_at:
                published_at = m_time.group(1)

            # Stats — look for patterns like "22赞" or just numbers near like/comment/share
            m_likes = re.search(r'["""]?(\d+(?:\.\d+)?万?)\s*赞', line)
            if m_likes:
                likes = parse_wan_number(m_likes.group(1))

            m_comments = re.search(r'["""]?(\d+(?:\.\d+)?万?)\s*评论', line)
            if m_comments:
                comments = parse_wan_number(m_comments.group(1))

            m_favs = re.search(r'["""]?(\d+(?:\.\d+)?万?)\s*收藏', line)
            if m_favs:
                favorites = parse_wan_number(m_favs.group(1))

            m_shares = re.search(r'["""]?(\d+(?:\.\d+)?万?)\s*分享', line)
            if m_shares:
                shares = parse_wan_number(m_shares.group(1))

            # Chapter summaries — look for timestamp patterns "00:00"
            m_chapter = re.search(r'^-?\s*(?:text:?\s*)?(\d{1,2}:\d{2})\s+(.+)', line)
            if m_chapter:
                ts = m_chapter.group(1)
                chapter_title = m_chapter.group(2).strip()
                # Next line(s) may contain the summary
                summary_lines = []
                j = i + 1
                while j < len(lines):
                    next_line = lines[j].strip()
                    if not next_line:
                        j += 1
                        continue
                    # Stop if next chapter or non-paragraph content
                    if re.search(r'^\d{1,2}:\d{2}\s', next_line):
                        break
                    if re.match(r'^-?\s*(?:text:?\s*)?(\d{1,2}:\d{2})', next_line):
                        break
                    if next_line.startswith(('- img', '- link', '- heading', "- button")):
                        break
                    # Collect paragraph text
                    clean = re.sub(r'^-?\s*(?:paragraph:?\s*|text:?\s*)', '', next_line).strip()
                    if clean:
                        summary_lines.append(clean)
                    j += 1
                chapters.append({
                    "timestamp": ts,
                    "title": chapter_title,
                    "summary": " ".join(summary_lines) if summary_lines else "",
                })

            # Description — long text blocks (not chapter content)
            if 'paragraph' in line.lower() or (line.startswith('- text:') and len(line) > 80):
                desc_text = re.sub(r'^-?\s*(?:paragraph:?\s*|text:?\s*)', '', line).strip()
                if len(desc_text) > len(description):
                    description = desc_text

            i += 1

        return {
            "url": url,
            "platform": "douyin",
            "title": title,
            "author": author,
            "description": description,
            "published_at": published_at,
            "stats": {
                "likes": likes,
                "comments": comments,
                "favorites": favorites,
                "shares": shares,
            },
            "chapters": chapters,
        }

    def to_markdown(self, data: Dict[str, Any]) -> str:
        parts = [f"# {data.get('title', 'Douyin Video')}\n"]
        if data.get('author'):
            parts.append(f"**作者**: {data['author']}")
        if data.get('published_at'):
            parts.append(f"**发布时间**: {data['published_at']}")

        stats = data.get('stats', {})
        stats_parts = []
        if stats.get('likes'): stats_parts.append(f"👍 {stats['likes']}")
        if stats.get('comments'): stats_parts.append(f"💬 {stats['comments']}")
        if stats.get('favorites'): stats_parts.append(f"⭐ {stats['favorites']}")
        if stats.get('shares'): stats_parts.append(f"🔄 {stats['shares']}")
        if stats_parts:
            parts.append(" | ".join(stats_parts))

        if data.get('description'):
            parts.append(f"\n## 描述\n\n{data['description']}")

        chapters = data.get('chapters', [])
        if chapters:
            parts.append("\n## 章节摘要\n")
            for ch in chapters:
                parts.append(f"**{ch['timestamp']}** {ch['title']}")
                if ch.get('summary'):
                    parts.append(f"> {ch['summary']}\n")

        parts.append(f"\n---\n*来源: {data.get('url', '')}*")
        return "\n".join(parts)


# ---------------------------------------------------------------------------
# Parser registry
# ---------------------------------------------------------------------------

PARSERS = [
    WeiboParser(),
    BilibiliParser(),
    CSDNParser(),
    WeixinParser(),
    DouyinParser(),
]


def get_parser(url: str) -> Optional[PlatformParser]:
    """Get appropriate parser for URL."""
    for parser in PARSERS:
        if parser.can_handle(url):
            return parser
    return None


# ---------------------------------------------------------------------------
# Main fetch function
# ---------------------------------------------------------------------------

def fetch(url: str, port: int = 9377) -> Dict[str, Any]:
    """Fetch content from any supported platform."""
    platform = identify_platform(url)
    if not platform:
        return {"url": url, "error": t("url_not_supported")}

    parser = get_parser(url)
    if not parser:
        return {"url": url, "error": t("platform_unsupported", platform=platform)}

    return parser.fetch(url, port)


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main():
    global _lang

    parser = argparse.ArgumentParser(
        description=(
            "Fetch posts from Chinese platforms (Weibo, Bilibili, CSDN).\n"
            "  --url <URL>    Platform URL to fetch\n"
            "  --pretty      Pretty print JSON\n"
            "  --text-only   Human-readable output\n"
            "  --markdown    Markdown output with YAML frontmatter\n"
            "  --port        Camofox port (default: 9377)\n"
            "  --lang        Language: zh (default) or en"
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--url", "-u", required=True, help="URL to fetch")
    parser.add_argument("--pretty", "-p", action="store_true", help="Pretty print JSON")
    parser.add_argument("--text-only", "-t", action="store_true", help="Human-readable output")
    parser.add_argument("--markdown", "-m", action="store_true", help="Markdown output with YAML frontmatter")
    parser.add_argument("--port", type=int, default=9377, help="Camofox port (default: 9377)")
    parser.add_argument(
        "--lang", default="zh", choices=["zh", "en"],
        help="Output language: zh (default) or en",
    )

    args = parser.parse_args()

    # Apply language setting
    _lang = args.lang

    indent = 2 if args.pretty else None

    # Fetch content
    result = fetch(args.url, port=args.port)

    # Output
    platform_parser = get_parser(args.url)
    if args.markdown:
        if platform_parser and "error" not in result:
            print(platform_parser.to_markdown(result))
        else:
            print(f"# Error\n{result.get('error', 'Unknown error')}", file=sys.stderr)
            sys.exit(1)
    elif args.text_only:
        if platform_parser and "error" not in result:
            print(platform_parser.to_text(result))
        else:
            print(f"Error: {result.get('error', 'Unknown error')}", file=sys.stderr)
            sys.exit(1)
    else:
        print(json.dumps(result, ensure_ascii=False, indent=indent))

    if result.get("error"):
        sys.exit(1)


if __name__ == "__main__":
    main()
