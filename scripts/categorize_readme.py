#!/usr/bin/env python3
"""Categorize README projects via OpenRouter and update README.md in place.

Requires environment variables:
  OPENROUTER_API_KEY  — API key (never logged)
  OPENROUTER_MODEL    — model id (e.g. google/gemma-4-31b-it:free)

Fails safely: never overwrites README.md unless the AI response is valid JSON
covering every project that still needs categorization.
"""

from __future__ import annotations

import json
import os
import re
import sys
import time
import urllib.error
import urllib.request
from typing import Any

OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions"
CATEGORIES_CACHE_PATH = os.environ.get("CATEGORIZE_CACHE_PATH", "categories.json")

# Free-tier OpenRouter models are often rate-limited (HTTP 429). Keep batches
# modest, space requests, and retry with backoff.
BATCH_SIZE = int(os.environ.get("CATEGORIZE_BATCH_SIZE", "25"))
BATCH_DELAY_SECONDS = float(os.environ.get("CATEGORIZE_BATCH_DELAY_SECONDS", "8"))
MAX_RETRIES = int(os.environ.get("CATEGORIZE_MAX_RETRIES", "8"))
RETRY_BASE_SECONDS = float(os.environ.get("CATEGORIZE_RETRY_BASE_SECONDS", "20"))

# Keep taxonomy small, reusable, and aligned with this AI/ML/DL collection.
ALLOWED_CATEGORIES = [
    "LLM",
    "Generative AI",
    "Machine Learning",
    "Deep Learning",
    "Computer Vision",
    "NLP",
    "AI Agents",
    "Coding Agents",
    "RAG",
    "MLOps",
    "AI Infrastructure",
    "Data Science",
    "Speech / Audio",
    "Multimodal AI",
    "MCP",
    "AI Tools",
    "Other",
]

PROJECT_RE = re.compile(
    r"^-\s+\[(?P<name>[^\]]+)\]\((?P<url>https://github\.com/[^)]+)\)"
    r"(?:\s+⭐\s+(?P<stars>\d+))?"
    r"(?:\s+—\s+(?P<desc>.*))?$"
)
CATEGORIES_SUFFIX_RE = re.compile(r"\s*·\s*\*\*Categories:\*\*\s*.*$")


def die(message: str, code: int = 1) -> None:
    print(f"error: {message}", file=sys.stderr)
    raise SystemExit(code)


def strip_categories(description: str | None) -> str:
    if not description:
        return ""
    return CATEGORIES_SUFFIX_RE.sub("", description).rstrip()


def extract_existing_categories(description: str | None) -> list[str]:
    if not description:
        return []
    match = re.search(r"\*\*Categories:\*\*\s*(.+)$", description)
    if not match:
        return []
    raw = match.group(1)
    return [c.strip() for c in re.findall(r"`([^`]+)`", raw) if c.strip()]


def format_categories(categories: list[str]) -> str:
    return " · ".join(f"`{c}`" for c in categories)


def parse_projects(readme: str) -> list[dict[str, Any]]:
    projects: list[dict[str, Any]] = []
    for line_no, line in enumerate(readme.splitlines(), start=1):
        match = PROJECT_RE.match(line)
        if not match:
            continue
        desc = match.group("desc")
        projects.append(
            {
                "line_no": line_no,
                "name": match.group("name"),
                "url": match.group("url"),
                "stars": match.group("stars"),
                "description": strip_categories(desc),
                "existing_categories": extract_existing_categories(desc),
                "raw_line": line,
            }
        )
    return projects


def extract_json_object(text: str) -> Any:
    """Parse JSON from model output that may include fences or prose."""
    text = text.strip()
    if not text:
        raise ValueError("empty model response")

    fence = re.search(r"```(?:json)?\s*([\s\S]*?)\s*```", text, re.IGNORECASE)
    if fence:
        text = fence.group(1).strip()

    try:
        return json.loads(text)
    except json.JSONDecodeError:
        pass

    start = text.find("{")
    end = text.rfind("}")
    if start == -1 or end == -1 or end <= start:
        raise ValueError("no JSON object found in model response")

    candidate = text[start : end + 1]
    try:
        return json.loads(candidate)
    except json.JSONDecodeError:
        # Mild repair: trailing commas before } or ]
        repaired = re.sub(r",\s*([}\]])", r"\1", candidate)
        return json.loads(repaired)


def normalize_categories(raw: Any) -> list[str]:
    if not isinstance(raw, list) or not raw:
        return []
    allowed = {c.lower(): c for c in ALLOWED_CATEGORIES}
    out: list[str] = []
    seen: set[str] = set()
    for item in raw:
        if not isinstance(item, str):
            continue
        key = item.strip()
        if not key:
            continue
        canonical = allowed.get(key.lower(), key.strip())
        # Map near-misses into allowed set when possible; otherwise keep concise custom labels.
        if canonical.lower() in allowed:
            canonical = allowed[canonical.lower()]
        elif canonical not in ALLOWED_CATEGORIES:
            # Prefer collapsing unknown labels into Other rather than inventing noise.
            canonical = "Other"
        if canonical not in seen:
            seen.add(canonical)
            out.append(canonical)
    return out or ["Other"]


def validate_projects_payload(
    data: Any, expected_names: set[str]
) -> dict[str, list[str]]:
    if not isinstance(data, dict) or "projects" not in data:
        raise ValueError("response JSON must be an object with a 'projects' array")
    projects = data["projects"]
    if not isinstance(projects, list) or not projects:
        raise ValueError("'projects' must be a non-empty array")

    result: dict[str, list[str]] = {}
    for item in projects:
        if not isinstance(item, dict):
            continue
        name = item.get("name")
        if not isinstance(name, str) or not name.strip():
            continue
        name = name.strip()
        cats = normalize_categories(item.get("categories"))
        if cats:
            result[name] = cats

    missing = expected_names - set(result)
    if missing:
        sample = ", ".join(sorted(missing)[:8])
        more = "" if len(missing) <= 8 else f" (+{len(missing) - 8} more)"
        raise ValueError(
            f"incomplete categorization; missing {len(missing)} projects: {sample}{more}"
        )

    return {name: result[name] for name in expected_names}


def build_prompt(batch: list[dict[str, Any]]) -> str:
    payload = [
        {
            "name": p["name"],
            "description": p["description"],
            "url": p["url"],
        }
        for p in batch
    ]
    categories_csv = ", ".join(ALLOWED_CATEGORIES)
    return f"""You are categorizing GitHub projects for an AI/ML/DL awesome list.

Categorize EVERY project in the input. Do not skip any.
Do not modify project names or descriptions.
Assign one or more relevant categories per project.
Use ONLY these categories (concise, reusable, consistent): {categories_csv}

Return ONLY valid JSON with this exact structure and no other text:
{{
  "projects": [
    {{
      "name": "owner/repo",
      "categories": ["LLM", "AI Agents"]
    }}
  ]
}}

Use each project's "name" field exactly as given.

Projects:
{json.dumps(payload, ensure_ascii=False, indent=2)}
"""


def _retry_after_seconds(exc: urllib.error.HTTPError, attempt: int) -> float:
    """Prefer Retry-After when present; otherwise exponential backoff."""
    header = exc.headers.get("Retry-After") if exc.headers else None
    if header:
        try:
            return max(float(header), 1.0)
        except ValueError:
            pass
    # Cap wait so a single Actions job does not hang forever.
    return min(RETRY_BASE_SECONDS * (2**attempt), 180.0)


def openrouter_chat(api_key: str, model: str, prompt: str) -> str:
    body = {
        "model": model,
        "messages": [
            {
                "role": "system",
                "content": (
                    "You categorize GitHub repositories. "
                    "Respond with ONLY valid JSON. No markdown fences. No commentary."
                ),
            },
            {"role": "user", "content": prompt},
        ],
        "temperature": 0.1,
    }
    data = json.dumps(body).encode("utf-8")

    last_error = "unknown error"
    for attempt in range(MAX_RETRIES + 1):
        request = urllib.request.Request(
            OPENROUTER_URL,
            data=data,
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
                "HTTP-Referer": "https://github.com/mparvin/awesome-stars",
                "X-Title": "awesome-stars-categorize",
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=180) as response:
                raw = response.read().decode("utf-8")
        except urllib.error.HTTPError as exc:
            # Never log response bodies — they can include request metadata.
            last_error = f"OpenRouter HTTP {exc.code}"
            if exc.code in {429, 502, 503} and attempt < MAX_RETRIES:
                wait_s = _retry_after_seconds(exc, attempt)
                print(
                    f"{last_error}; rate-limited/unavailable. "
                    f"Retry {attempt + 1}/{MAX_RETRIES} in {wait_s:.0f}s...",
                    flush=True,
                )
                time.sleep(wait_s)
                continue
            die(f"{last_error}: request failed")
        except urllib.error.URLError as exc:
            last_error = f"OpenRouter request failed: {exc.reason}"
            if attempt < MAX_RETRIES:
                wait_s = min(RETRY_BASE_SECONDS * (2**attempt), 180.0)
                print(
                    f"{last_error}. Retry {attempt + 1}/{MAX_RETRIES} in {wait_s:.0f}s...",
                    flush=True,
                )
                time.sleep(wait_s)
                continue
            die(last_error)

        try:
            parsed = json.loads(raw)
        except json.JSONDecodeError:
            die("OpenRouter returned non-JSON response")

        # Some providers return 200 with an error object when rate-limited.
        err_obj = parsed.get("error") if isinstance(parsed, dict) else None
        if isinstance(err_obj, dict):
            code = err_obj.get("code") or err_obj.get("type") or "error"
            # Do not print error message text — may contain sensitive details.
            last_error = f"OpenRouter API error ({code})"
            code_str = str(code).lower()
            if (
                "rate" in code_str or code in {429, "429"} or "quota" in code_str
            ) and attempt < MAX_RETRIES:
                wait_s = min(RETRY_BASE_SECONDS * (2**attempt), 180.0)
                print(
                    f"{last_error}; retry {attempt + 1}/{MAX_RETRIES} in {wait_s:.0f}s...",
                    flush=True,
                )
                time.sleep(wait_s)
                continue
            die(last_error)

        try:
            content = parsed["choices"][0]["message"]["content"]
        except (KeyError, IndexError, TypeError):
            die("OpenRouter response missing choices[0].message.content")

        if not isinstance(content, str) or not content.strip():
            die("OpenRouter returned empty message content")
        return content

    die(last_error)


def categorize_projects(
    projects: list[dict[str, Any]],
    api_key: str,
    model: str,
    *,
    on_batch_done: Any | None = None,
) -> dict[str, list[str]]:
    merged: dict[str, list[str]] = {}
    total_batches = (len(projects) + BATCH_SIZE - 1) // BATCH_SIZE
    for batch_idx, i in enumerate(range(0, len(projects), BATCH_SIZE)):
        if batch_idx > 0 and BATCH_DELAY_SECONDS > 0:
            print(
                f"Waiting {BATCH_DELAY_SECONDS:.0f}s before next batch "
                f"to respect rate limits...",
                flush=True,
            )
            time.sleep(BATCH_DELAY_SECONDS)

        batch = projects[i : i + BATCH_SIZE]
        names = {p["name"] for p in batch}
        print(
            f"Categorizing batch {batch_idx + 1}/{total_batches} "
            f"({len(batch)} projects)...",
            flush=True,
        )
        content = openrouter_chat(api_key, model, build_prompt(batch))
        try:
            data = extract_json_object(content)
            batch_map = validate_projects_payload(data, names)
        except (ValueError, json.JSONDecodeError) as exc:
            die(
                f"failed to parse/validate AI response for batch "
                f"{batch_idx + 1}/{total_batches}: {exc}"
            )
        merged.update(batch_map)
        if on_batch_done is not None:
            on_batch_done(merged)
    return merged


def apply_categories(readme: str, category_map: dict[str, list[str]]) -> str:
    lines = readme.splitlines(keepends=True)
    out: list[str] = []
    for line in lines:
        newline = ""
        core = line
        if line.endswith("\r\n"):
            newline = "\r\n"
            core = line[:-2]
        elif line.endswith("\n"):
            newline = "\n"
            core = line[:-1]

        match = PROJECT_RE.match(core)
        if not match:
            out.append(line)
            continue

        name = match.group("name")
        cats = category_map.get(name)
        if not cats:
            out.append(line)
            continue

        stars = match.group("stars")
        desc = strip_categories(match.group("desc"))
        rebuilt = f"- [{name}]({match.group('url')})"
        if stars:
            rebuilt += f" ⭐ {stars}"
        if desc:
            rebuilt += f" — {desc}"
        rebuilt += f" · **Categories:** {format_categories(cats)}"
        out.append(rebuilt + newline)

    return "".join(out)


def load_categories_cache(path: str) -> dict[str, list[str]]:
    if not os.path.exists(path):
        return {}
    try:
        with open(path, encoding="utf-8") as f:
            data = json.load(f)
    except (OSError, json.JSONDecodeError) as exc:
        print(f"warning: ignoring unreadable cache {path}: {exc}", flush=True)
        return {}

    projects = data.get("projects") if isinstance(data, dict) else None
    if not isinstance(projects, list):
        return {}

    out: dict[str, list[str]] = {}
    for item in projects:
        if not isinstance(item, dict):
            continue
        name = item.get("name")
        if not isinstance(name, str) or not name.strip():
            continue
        cats = normalize_categories(item.get("categories"))
        if cats:
            out[name.strip()] = cats
    return out


def save_categories_cache(path: str, category_map: dict[str, list[str]]) -> None:
    payload = {
        "projects": [
            {"name": name, "categories": cats}
            for name, cats in sorted(category_map.items())
        ]
    }
    try:
        with open(path, "w", encoding="utf-8") as f:
            json.dump(payload, f, ensure_ascii=False, indent=2)
            f.write("\n")
    except OSError as exc:
        die(f"cannot write {path}: {exc}")


def write_readme_safely(
    readme_path: str,
    original: str,
    category_map: dict[str, list[str]],
    expected_count: int,
) -> bool:
    """Apply categories and write README. Returns True if file changed."""
    updated = apply_categories(original, category_map)
    if updated == original:
        return False
    if len(parse_projects(updated)) != expected_count:
        die("refusing to write README: project count changed after applying categories")
    try:
        with open(readme_path, "w", encoding="utf-8") as f:
            f.write(updated)
    except OSError as exc:
        die(f"cannot write {readme_path}: {exc}")
    return True


def main() -> None:
    readme_path = sys.argv[1] if len(sys.argv) > 1 else "README.md"
    cache_path = CATEGORIES_CACHE_PATH

    try:
        with open(readme_path, encoding="utf-8") as f:
            original = f.read()
    except OSError as exc:
        die(f"cannot read {readme_path}: {exc}")

    projects = parse_projects(original)
    if not projects:
        die(f"no projects found in {readme_path}")

    cached = load_categories_cache(cache_path)
    category_map: dict[str, list[str]] = {}
    needs_api: list[dict[str, Any]] = []

    for project in projects:
        name = project["name"]
        if project["existing_categories"]:
            category_map[name] = normalize_categories(project["existing_categories"])
        elif name in cached:
            category_map[name] = cached[name]
        else:
            needs_api.append(project)

    print(
        f"Found {len(projects)} projects; "
        f"{len(category_map)} already categorized; "
        f"{len(needs_api)} need API categorization."
    )
    print(
        f"Batch size={BATCH_SIZE}, inter-batch delay={BATCH_DELAY_SECONDS:.0f}s, "
        f"max retries={MAX_RETRIES}.",
        flush=True,
    )

    def checkpoint(progress: dict[str, list[str]]) -> None:
        save_categories_cache(cache_path, progress)
        if write_readme_safely(readme_path, original, progress, len(projects)):
            print(
                f"Checkpoint: saved categories for {len(progress)}/{len(projects)} projects.",
                flush=True,
            )

    if needs_api:
        api_key = os.environ.get("OPENROUTER_API_KEY", "").strip()
        model = os.environ.get("OPENROUTER_MODEL", "").strip()
        if not api_key:
            die("OPENROUTER_API_KEY is required")
        if not model:
            die("OPENROUTER_MODEL is required")
        print(f"Using OpenRouter model from OPENROUTER_MODEL ({len(model)} chars).")

        def on_batch_done(fetched: dict[str, list[str]]) -> None:
            progress = dict(category_map)
            progress.update(fetched)
            checkpoint(progress)

        fetched = categorize_projects(
            needs_api, api_key, model, on_batch_done=on_batch_done
        )
        category_map.update(fetched)

    if len(category_map) != len(projects):
        # Keep whatever we have so the next run only asks for the remainder.
        checkpoint(category_map)
        die(
            f"incomplete categorization: have {len(category_map)}/{len(projects)} projects. "
            "Re-run later to finish remaining projects."
        )

    save_categories_cache(cache_path, category_map)
    if write_readme_safely(readme_path, original, category_map, len(projects)):
        print(f"Updated {readme_path} with categories for {len(projects)} projects.")
    else:
        print("README.md already categorized; no changes.")


if __name__ == "__main__":
    main()
