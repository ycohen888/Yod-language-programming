# -*- coding: utf-8 -*-
import json
import re
from pathlib import Path

root = Path(__file__).resolve().parent.parent / "עמודים"
index = {}
for p in sorted(root.glob("*.html")):
    href = "עמודים/" + p.name
    text = p.read_text(encoding="utf-8")
    secs = []
    for m in re.finditer(r"<h([23])([^>]*)>(.*?)</h\1>", text, re.S | re.I):
        level = m.group(1)
        attrs = m.group(2)
        inner = re.sub(r"<[^>]+>", "", m.group(3))
        title = re.sub(r"\s+", " ", inner).strip()
        if not title or len(title) > 100:
            continue
        idm = re.search(r"""id=["']([^"']+)["']""", attrs)
        sid = idm.group(1) if idm else ""
        if not sid:
            raw = re.sub(r"[^\w\u0590-\u05FF\-]+", "", title.replace(" ", "-"))
            sid = "s-" + raw if raw else ""
        secs.append({"title": title, "id": sid, "level": "h" + level})
    seen = set()
    out = []
    for s in secs:
        if s["title"] in seen:
            continue
        seen.add(s["title"])
        out.append(s)
    if out:
        index[href] = out

js_path = Path(__file__).resolve().parent / "guide-sections.js"
payload = json.dumps(index, ensure_ascii=False)
js_path.write_text(
    "/* נוצר אוטומטית מ־gen_sections.py — אינדקס כותרות משנה לחיפוש */\n"
    "window.GUIDE_SECTIONS = " + payload + ";\n",
    encoding="utf-8",
)
print("wrote", js_path, "pages", len(index), "sections", sum(len(v) for v in index.values()))
