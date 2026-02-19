import json
import re
import sys
import unicodedata
from typing import Any, Dict, List, Optional


SLANG_WORDS = {"awesome", "cool", "kinda", "sorta", "gonna", "wanna", "lol"}
SHORTER_TOO_LONG_THRESHOLD = 320
SHORTER_SUGGESTED_LIMIT = 180
SIMPLIFY_SUGGESTED_LIMIT = 200
SIMPLIFY_LONG_WORD_MIN = 16
SIMPLIFY_LONG_WORD_COUNT = 2


def _is_bullet_line(line: str) -> bool:
    return bool(re.match(r"^\s*(?:[-*•]|\d+[.)])\s+", line))


def _to_bullets(text: str) -> str:
    parts = [p.strip() for p in re.split(r"(?<=[.!?])\s+", text.strip()) if p.strip()]
    if not parts:
        parts = [text.strip()] if text.strip() else []
    return "\n".join(f"- {p}" for p in parts)


def _to_paragraph(text: str) -> str:
    lines = [re.sub(r"^\s*(?:[-*•]|\d+[.)])\s+", "", line).strip() for line in text.splitlines()]
    return " ".join(line for line in lines if line)


def _sanitize_professional(text: str) -> str:
    clean = "".join(ch for ch in text if unicodedata.category(ch) != "So")
    words = []
    for word in clean.split():
        plain = re.sub(r"[^a-zA-Z]", "", word).lower()
        if plain in SLANG_WORDS:
            continue
        words.append(word)
    clean = " ".join(words)
    clean = re.sub(r"!{2,}", "!", clean)
    return clean.strip()


def _contains_symbol_emoji(text: str) -> bool:
    return any(unicodedata.category(ch) == "So" for ch in text)


def _shorten(text: str, limit: int = 180) -> str:
    text = text.strip()
    if len(text) <= limit:
        return text
    shortened = text[: limit - 1].rsplit(" ", 1)[0].strip()
    return (shortened or text[: limit - 1]).rstrip(" ,;:-") + "…"


def review(payload: Dict[str, Any]) -> Dict[str, Any]:
    request_id = payload.get("requestId", "")
    output = str(payload.get("output", "")).strip()
    mode = payload.get("mode")
    style = payload.get("style")

    issues: List[str] = []
    major = False

    lines = [line for line in output.splitlines() if line.strip()]
    bullet_lines = [line for line in lines if _is_bullet_line(line)]

    if style == "bullet" and lines and len(bullet_lines) != len(lines):
        issues.append("Style mismatch: expected bullet format.")
        major = True
    elif style == "paragraph" and bullet_lines:
        issues.append("Style mismatch: expected paragraph format.")
        major = True

    if mode == "shorter" and len(output) > SHORTER_TOO_LONG_THRESHOLD:
        issues.append("Too long for shorter mode.")
        major = True

    if mode == "professional":
        tokens = re.findall(r"\b\w+\b", output.lower())
        if any(token in SLANG_WORDS for token in tokens) or _contains_symbol_emoji(output):
            issues.append("Tone mismatch: expected professional language.")
            major = True

    if mode == "simplify":
        long_words = [w for w in re.findall(r"\b\w+\b", output) if len(w) >= SIMPLIFY_LONG_WORD_MIN]
        if len(long_words) >= SIMPLIFY_LONG_WORD_COUNT:
            issues.append("Mode mismatch: simplify output is too complex.")
            major = True

    if not issues:
        return {
            "requestId": request_id,
            "pass": True,
            "issues": [],
            "suggestedEdit": None,
        }

    if not major:
        return {
            "requestId": request_id,
            "pass": True,
            "issues": issues,
            "suggestedEdit": None,
        }

    suggested = output
    if mode == "professional":
        suggested = _sanitize_professional(suggested)
    if mode == "simplify":
        suggested = _shorten(suggested, SIMPLIFY_SUGGESTED_LIMIT)
    if mode == "shorter":
        suggested = _shorten(suggested, SHORTER_SUGGESTED_LIMIT)
    if style == "bullet":
        suggested = _to_bullets(_to_paragraph(suggested))
    elif style == "paragraph":
        suggested = _to_paragraph(suggested)

    return {
        "requestId": request_id,
        "pass": False,
        "issues": issues,
        "suggestedEdit": suggested or None,
    }


def main() -> None:
    try:
        payload = json.load(sys.stdin)
    except json.JSONDecodeError:
        print(json.dumps({"requestId": "", "pass": False, "issues": ["Invalid JSON input."], "suggestedEdit": None}))
        return

    result = review(payload)
    print(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()
