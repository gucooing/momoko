"""Export 学マス (gakumas) client textures as WebP for the Momoko frontend.

Images are static frontend files: this script is run offline (after a game update)
and its output is committed. They are NOT part of Vite's module graph, so nothing is
loaded up front: a page loads the small name -> hash index of one group only when it
first shows an image of that group, and the browser fetches only the images it shows.
File names carry a content hash, so they can be cached forever. Momoko never fetches
images from the game server.

Usage (Windows example):
  pip install UnityPy Pillow
  python frontend/scripts/games/gakumas/extract-assets.py ^
      --resource C:/Users/gucooing/Desktop/gakumas/resource ^
      --codec    C:/Users/gucooing/Desktop/gakumas/client/tools
  # only some groups:      --only chara,cards/idol
  # also export meishi:     --with-meishi
  # re-export everything:  --force

Input layout (gakumas resource mirror):
  resource/manifest.json            {"version", "revision", "objects": [{"name", "md5", "size", ...}]}
  resource/blobs/<md5[:2]>/<md5>    one UnityFS bundle per object; the first 256
                                    bytes are masked with resource_codec.name_mask(name)

Output (defaults are relative to the frontend directory):
  public/games/gakumas/<group>/<name>.<hash>.webp   e.g. cards/idol/hski-3-000-sq.1a2b3c4d.webp
  src/games/gakumas/assets/<group>.json             {"<name>": "<hash>"}; "/" in the group
                                                    becomes "-" (cards/idol -> cards-idol.json)
  scripts/games/gakumas/state.json                  {"<group>/<name>": {"w", "h", "src", "md5", "hash"}}
                                                    (incremental export state, not shipped)
"""
from __future__ import annotations

import argparse
import glob
import hashlib
import io
import json
import os
import re
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

# (group, regex on manifest object name, key template, (max_w, max_h or None))
RULES = [
    ("chara", r"^img_chr_(?P<c>[a-z]+)_00-thumb-circle$", "{c}", (128, 128)),
    ("chara", r"^img_chr_(?P<c>[a-z]+)_00-bustup$", "{c}-bust", (512, None)),
    ("cards/idol", r"^img_general_cidol-(?P<id>[a-z]+-\d-\d{3})_1-thumb-square$", "{id}-sq", (256, 256)),
    ("cards/idol", r"^img_general_cidol-(?P<id>[a-z]+-\d-\d{3})_1-thumb-landscape-large$", "{id}-land", (960, 540)),
    ("cards/support", r"^img_general_csprt-(?P<id>\d-\d{4})_thumb-square$", "{id}-sq", (256, 256)),
    ("cards/support", r"^img_general_csprt-(?P<id>\d-\d{4})_thumb-landscape$", "{id}-land", (960, 540)),
    ("gasha", r"^img_gasha_banner_(?P<id>gasha[-_].+)-select-m$", "{id}", (920, 320)),
    ("gasha", r"^img_general_gasha_bg_(?P<id>.+)$", "bg-{id}", (1280, None)),
    ("items", r"^img_general_item_(?P<id>.+)$", "{id}", (96, 96)),
    ("events", r"^img_general_event_(?P<t>[a-z-]+)_(?P<id>.+)-(?P<p>banner|logo)$", "{id}-{p}", (1024, None)),
    ("events", r"^img_notice_gasha_(?P<id>gasha-.+)-banner-01-large$", "notice-{id}", (1024, None)),
    ("ui", r"^img_login_title_(?P<id>.+)$", "login-{id}", (1024, None)),
]
MEISHI = [("ui", r"^img_general_meishi_(?P<k>base|illust)_(?P<id>.+)$", "meishi-{k}-{id}", (1024, None))]

FRONTEND = Path(__file__).resolve().parents[3]


def fit(w: int, h: int, max_w: int, max_h: int | None) -> tuple[int, int]:
    scale = min(1.0, max_w / w, (max_h / h) if max_h else 1.0)
    return max(1, round(w * scale)), max(1, round(h * scale))


def write_group_indexes(index: dict, out: Path) -> None:
    """One small {"<name>": "<hash>"} file per group, loaded lazily by the frontend."""
    groups: dict[str, dict[str, str]] = {}
    for key, meta in sorted(index.items()):
        if not meta.get("hash"):
            continue
        group, base = key.rsplit("/", 1)
        groups.setdefault(group, {})[base] = meta["hash"]
    out.mkdir(parents=True, exist_ok=True)
    for old in out.glob("*.json"):
        old.unlink()
    for group, names in groups.items():
        (out / f"{group.replace('/', '-')}.json").write_text(json.dumps(names, separators=(",", ":")), encoding="utf-8")


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--resource", type=Path, required=True, help="gakumas resource dir (manifest.json + blobs/)")
    ap.add_argument("--codec", type=Path, required=True, help="dir containing resource_codec.py")
    ap.add_argument("--out", type=Path, default=FRONTEND / "public/games/gakumas", help="image dir (served as static files)")
    ap.add_argument("--index", type=Path, default=FRONTEND / "src/games/gakumas/assets", help="per-group name -> hash indexes")
    ap.add_argument("--state", type=Path, default=Path(__file__).with_name("state.json"), help="incremental export state")
    ap.add_argument("--only", default="", help="comma separated groups, e.g. chara,cards/idol")
    ap.add_argument("--with-meishi", action="store_true", help="also export meishi images (2,000+ files)")
    ap.add_argument("--quality", type=int, default=82)
    ap.add_argument("--force", action="store_true", help="re-export even if md5 is unchanged")
    ap.add_argument("--jobs", type=int, default=os.cpu_count() or 4, help="parallel exports (WebP encoding dominates)")
    a = ap.parse_args()

    sys.path.insert(0, str(a.codec))
    from resource_codec import unmask  # noqa: E402  (gakumas client tool)
    import UnityPy  # noqa: E402
    from PIL import Image  # noqa: E402

    UnityPy.config.FALLBACK_UNITY_VERSION = "2022.3.21f1"

    manifest = json.loads((a.resource / "manifest.json").read_text(encoding="utf-8"))
    rules = RULES + (MEISHI if a.with_meishi else [])
    only = {g.strip() for g in a.only.split(",") if g.strip()}
    compiled = [(g, re.compile(r), k, size) for g, r, k, size in rules if not only or g in only]

    index_path = a.state
    index: dict = json.loads(index_path.read_text(encoding="utf-8")) if index_path.exists() else {}
    stats = {"ok": 0, "same": 0, "missing": 0, "error": 0}

    jobs = []
    for obj in manifest["objects"]:
        name, md5 = obj["name"], obj["md5"]
        for group, rx, key_tpl, (max_w, max_h) in compiled:
            m = rx.match(name)
            if not m:
                continue
            key = f"{group}/{key_tpl.format(**m.groupdict())}"
            prev = index.get(key, {})
            dst = a.out / f"{key}.{prev.get('hash', '')}.webp"
            if not a.force and prev.get("hash") and dst.exists() and prev.get("md5") == md5:
                stats["same"] += 1
                break
            blob = a.resource / "blobs" / md5[:2] / md5
            if not blob.exists():
                stats["missing"] += 1
                print("missing", name)
                break
            jobs.append((key, name, md5, blob, dst, max_w, max_h))
            break

    def export(key: str, name: str, md5: str, blob: Path, dst: Path, max_w: int, max_h: int | None) -> dict:
        raw = blob.read_bytes()
        data = raw if raw.startswith(b"UnityFS\0") else unmask(raw, name)
        env = UnityPy.load(data)
        textures = [o.read() for o in env.objects if o.type.name == "Texture2D"]
        if not textures:
            raise ValueError("no Texture2D")
        tex = max(textures, key=lambda t: t.m_Width * t.m_Height)
        img = tex.image.convert("RGBA")
        w, h = fit(img.width, img.height, max_w, max_h)
        if (w, h) != img.size:
            img = img.resize((w, h), Image.LANCZOS)
        buf = io.BytesIO()
        img.save(buf, "WEBP", quality=a.quality, method=6)
        body = buf.getvalue()
        digest = hashlib.sha256(body).hexdigest()[:8]
        group, base = key.rsplit("/", 1)
        folder = a.out / group
        folder.mkdir(parents=True, exist_ok=True)
        for old in folder.glob(f"{glob.escape(base)}.*.webp"):
            if old.name.count(".") == 2:  # <name>.<hash>.webp of an earlier export
                old.unlink()
        (folder / f"{base}.{digest}.webp").write_bytes(body)
        return {"w": w, "h": h, "src": name, "md5": md5, "hash": digest}

    with ThreadPoolExecutor(max(1, a.jobs)) as pool:
        futures = {pool.submit(export, *job): job for job in jobs}
        for f in as_completed(futures):
            key, name = futures[f][:2]
            try:
                index[key] = f.result()
                stats["ok"] += 1
                print("ok", key, f"{index[key]['w']}x{index[key]['h']}")
            except Exception as e:  # keep going; report at the end
                stats["error"] += 1
                print("error", name, e)

    index_path.write_text(json.dumps(dict(sorted(index.items())), ensure_ascii=False, indent=1), encoding="utf-8")
    write_group_indexes(index, a.index)
    total = sum(p.stat().st_size for p in a.out.rglob("*.webp"))
    print(f"done: {stats} · {len(index)} files · {total / 1e6:.1f} MB · manifest version {manifest.get('version')} rev {manifest.get('revision')}")
    return 1 if stats["error"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
