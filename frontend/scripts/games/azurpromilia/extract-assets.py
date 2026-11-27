"""Export 蓝色星原 (Azur Promilia, TC client) icons and key art as WebP for the Momoko frontend.

Same static-asset system as gakumas (see frontend/src/games/core.ts): this script is run offline
(after a game update) and its output is committed. Images are not part of Vite's module graph; a
page loads the small id -> hash index of one group only when it first shows an image of that
group. File names carry a content hash. Momoko never fetches images from the game server.

Which image belongs to which ID follows what the client reads, not file names. Table rows come
from the server res export (Excel/*.json); the paths in them are client paths such as
UI/Icon/Item/x.png (UI/Icon/* lives under Assets/Arts/, UI/Texture/* under Assets/ResourcesAssets/).

Groups and keys:
  heroes/<hero id>       hero_clothing[hero id].avatarTexture[2] (UI/Icon/HeroS: the hero list icon and
                         the reward icon of type 1, RewardUtil.GetRewardIcon)
  pets/<pet id>          pet.petIcon[0] (reward type 5)
  items/<kind>/<id>      kind = the reward type's table (reward-types.json); RewardUtil.GetRewardIcon /
                         ItemManager.getItemConfig read:
    common_item, soulessence, currency, player_title, task_item, collection_tool, mount_saddle,
    playercard_dress, pet_skin, home_dorm_item, vehicle, chat_emoji, chat_bg, chat_bubble: icon[0]
    accessory: texture[0]    pet_egg: eggImg[0]    mount: icon (a plain string)
    home_building: home_object[objId].objIcon[0]
    social_wheel: icon[0] (not a reward icon; the social wheel cells read it)
    Reward type 29 (reputation) has no group of its own: the client reads the currency table for it.
  heads/<dress id>       player avatar of a 名片 head part: playercard_dress rows of type 101
                         (L_Const.playerDressType.head) with a non-empty profilepic -> profilepic[0]
                         (playerDisplayStore.getHeadPath / loadHeadPic)
  heads/default-<sex>    the default avatar (sex 1 male, 2 female): photoManager.getDefaultHeadPath("head4",
                         sex) = UI/Texture/Mod/default_<male|female>head4.png; "head4" is the screenshot
                         type of avatarTextureIndex.half_formation (getPlayerAvatarsFormationHead)
  app/icon               the game's app icon: the largest image of the icon resource of AzurPromilia.exe
                         (what Windows and the launcher show), 256 px
  covers/loading-<id>    loading-screen illustrations: tips_group.bg[0] (loadingManager / loadingBehavior_*),
                         id = the smallest tips_group id that uses the picture
  Gacha pool backgrounds (GachaComBg) are not covers: they are transparent layers drawn under the
  client's overlays, so on their own they show empty or cropped areas.
  Rows without an image, or whose image is not in the client's default_package, are listed at the end.

Sizes: icons fit into --size (128) px, heads into 256 px, both drawn at their full sprite rect
(transparent margin kept); covers at most 1280 px wide (aspect kept, trimmed sprite). WebP --quality 82.

Usage (Windows example). The resource tools of the Azur Promilia analysis project are reused for
manifests, VString path recovery and the UnityCN AssetBundle key; its tmp/resources/aes_*.bin
(MMap keys, written by tools/prepare_client_assets.py for this client binary) must exist. UnityPy,
Pillow and pycryptodome are the ones those tools import. Nothing is written to the game or tools dir.
  python -I -B frontend/scripts/games/azurpromilia/extract-assets.py ^
      --game  E:/game/AzurPromilia_TC/AzurPromilia_TC_game ^
      --tools E:/ps/AzurPromilia/tools ^
      --res   E:/ps/AzurPromilia/server/res
  # only some groups:      --only heroes,items/common_item,covers
  # re-export everything:  --force

Output (defaults are relative to the frontend directory):
  public/games/azurpromilia/<group>/<id>.<hash>.webp  e.g. items/common_item/9000.1a2b3c4d.webp
  src/games/azurpromilia/assets/<group>.json          {"<id>": "<hash>"}; "/" in the group becomes "-"
                                                      (items/common_item -> items-common_item.json)
  scripts/games/azurpromilia/state.json               {"<group>/<id>": {"w", "h", "src", "asset", "crc", "fit", "hash"}}
                                                      (incremental export state, not shipped; an image is
                                                      re-encoded when its path, bundle CRC or size changes)
"""
from __future__ import annotations

import argparse
import glob
import hashlib
import io
import json
import os
import struct
import sys
from collections import Counter, defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

sys.dont_write_bytecode = True  # never leave __pycache__ in the tools dir

FRONTEND = Path(__file__).resolve().parents[3]
MAX_SIZE = 128  # icons are shown at most ~64 px wide in Momoko; 128 covers 2x screens

# (kind, field, index): items/<kind>/<id> = row[field][index]; a plain string field is the path itself.
ITEM_ICONS = [
    ("common_item", "icon", 0),
    ("soulessence", "icon", 0),
    ("currency", "icon", 0),
    ("mount", "icon", 0),
    ("pet_egg", "eggImg", 0),
    ("accessory", "texture", 0),
    ("player_title", "icon", 0),
    ("task_item", "icon", 0),
    ("collection_tool", "icon", 0),
    ("mount_saddle", "icon", 0),
    ("playercard_dress", "icon", 0),
    ("pet_skin", "icon", 0),
    ("home_dorm_item", "icon", 0),
    ("vehicle", "icon", 0),
    ("chat_emoji", "icon", 0),
    ("chat_bg", "icon", 0),
    ("chat_bubble", "icon", 0),
    ("social_wheel", "icon", 0),
]

HEAD_DRESS_TYPE = 101  # L_Const.playerDressType.head
SEXES = {1: "male", 2: "female"}  # L_Const.sexType
DEFAULT_HEAD = "UI/Texture/Mod/default_{}head4.png"  # photoManager _defaultTexturePaths.head4

# group -> (max width, max height or None, keep the full sprite rect)
FITS = {"heads": (256, 256, True), "covers": (1280, None, False)}

APP_ICON = "app/icon"  # the game's app icon: the largest image of AzurPromilia.exe's icon resource
APP_EXE = "AzurPromilia.exe"
APP_ICON_SIZE = 256


def read_rows(res: Path, table: str) -> list[dict]:
    data = json.loads((res / "Excel" / f"{table}.json").read_text(encoding="utf-8"))
    return data["list"] if isinstance(data, dict) else data


def icon_at(value, index: int) -> str:
    if isinstance(value, str):
        return value
    if isinstance(value, list) and len(value) > index and isinstance(value[index], str):
        return value[index]
    return ""


def wanted_images(res: Path) -> dict[str, str]:
    """key -> client image path ('' when the row has none)."""
    out: dict[str, str] = {}
    clothing = {r["clothingid"]: r for r in read_rows(res, "hero_clothing")}
    for hero in read_rows(res, "hero"):
        out[f"heroes/{hero['id']}"] = icon_at(clothing.get(hero["id"], {}).get("avatarTexture"), 2)
    for pet in read_rows(res, "pet"):
        out[f"pets/{pet['id']}"] = icon_at(pet.get("petIcon"), 0)
    objects = {r["id"]: r for r in read_rows(res, "home_object")}
    for b in read_rows(res, "home_building"):
        out[f"items/home_building/{b['id']}"] = icon_at(objects.get(b["objId"], {}).get("objIcon"), 0)
    for kind, field, index in ITEM_ICONS:
        for row in read_rows(res, kind):
            out[f"items/{kind}/{row['id']}"] = icon_at(row.get(field), index)

    for row in read_rows(res, "playercard_dress"):
        if row.get("type") == HEAD_DRESS_TYPE:
            out[f"heads/{row['id']}"] = icon_at(row.get("profilepic"), 0)
    for sex, name in SEXES.items():
        out[f"heads/default-{sex}"] = DEFAULT_HEAD.format(name)

    loading: dict[str, int] = {}
    for row in read_rows(res, "tips_group"):
        bg = icon_at(row.get("bg"), 0)
        if bg:
            loading[bg] = min(loading.get(bg, row["id"]), row["id"])
    for bg, tip in loading.items():
        out[f"covers/loading-{tip}"] = bg
    return out


def asset_candidates(path: str, lang: str) -> list[str]:
    """Full asset paths a client path can load from, in the client's order (I18NManager.ConvertAssetPath:
    language variants first; UI/Icon under Assets/Arts, other UI textures under Assets/ResourcesAssets)."""
    if path.lower().startswith("assets/"):
        return [path]
    out = [f"Assets/ResourcesLang/{lang}/{path}"]
    if path.startswith("UI/Icon/"):
        out.append(f"Assets/Arts/UI/LangRes/{lang}/{path[len('UI/'):]}")
    return out + [f"Assets/Arts/{path}", f"Assets/ResourcesAssets/{path}"]


def group_of(key: str) -> str:
    return key.rsplit("/", 1)[0]


def fit(w: int, h: int, max_w: int, max_h: int | None) -> tuple[int, int]:
    scale = min(1.0, max_w / w, (max_h / h) if max_h else 1.0)
    return max(1, round(w * scale)), max(1, round(h * scale))


def sprite_image(sprite, Image, full_rect: bool):
    """The sprite image; with full_rect, at its full rect: UnityPy returns only the trimmed texture rect of
    a tight sprite, which would shift and rescale icons that the client draws inside a transparent margin."""
    img = sprite.image.convert("RGBA")
    rect, rd = sprite.m_Rect, sprite.m_RD
    w, h = round(rect.width), round(rect.height)
    if not full_rect or (w, h) == img.size or w <= 0 or h <= 0:
        return img
    canvas = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    x = round(rd.textureRectOffset.x)
    y = round(h - rd.textureRectOffset.y - rd.textureRect.height)  # Unity rects start bottom-left
    canvas.paste(img, (x, y))
    return canvas


def exe_icon(exe: Path) -> bytes:
    """The largest image of the first RT_GROUP_ICON of a PE file (the icon Windows shows for it), as a
    one-image .ico file (PNG-compressed images are returned as the PNG itself)."""
    data = exe.read_bytes()
    u16 = lambda o: struct.unpack_from("<H", data, o)[0]  # noqa: E731
    u32 = lambda o: struct.unpack_from("<I", data, o)[0]  # noqa: E731
    pe = u32(0x3C)
    if data[pe:pe + 4] != b"PE\0\0":
        raise SystemExit(f"{exe}: not a PE file")
    sections, opt_size = u16(pe + 6), u16(pe + 20)
    opt = pe + 24
    rsrc_rva = u32(opt + (112 if u16(opt) == 0x20B else 96) + 2 * 8)  # data directory 2: resources
    table = [(u32(s + 12), max(u32(s + 8), u32(s + 16)), u32(s + 20))
             for s in range(opt + opt_size, opt + opt_size + sections * 40, 40)]

    def offset(rva: int) -> int:
        for va, size, raw in table:
            if va <= rva < va + size:
                return rva - va + raw
        raise SystemExit(f"{exe}: RVA {rva:#x} is outside every section")

    root = offset(rsrc_rva)

    def entries(directory: int) -> list[tuple[int, int]]:
        n = u16(directory + 12) + u16(directory + 14)
        return [(u32(e), u32(e + 4)) for e in range(directory + 16, directory + 16 + n * 8, 8)]

    def first_data(entry: int) -> bytes:  # follow the first name / language down to the data
        while entry & 0x80000000:
            entry = entries(root + (entry & 0x7FFFFFFF))[0][1]
        rva, size = u32(root + entry), u32(root + entry + 4)
        return data[offset(rva):offset(rva) + size]

    types = dict(entries(root))
    if 3 not in types or 14 not in types:
        raise SystemExit(f"{exe}: no icon resource")
    icons = {name: off for name, off in entries(root + (types[3] & 0x7FFFFFFF))}
    group = first_data(types[14])
    count = struct.unpack_from("<H", group, 4)[0]
    best = max((group[6 + i * 14:6 + i * 14 + 14] for i in range(count)),
               key=lambda e: (e[0] or 256, struct.unpack_from("<H", e, 6)[0]))
    body = first_data(icons[struct.unpack_from("<H", best, 12)[0]])
    if body.startswith(b"\x89PNG"):
        return body
    return struct.pack("<HHH", 0, 1, 1) + best[:12] + struct.pack("<I", 22) + body


def write_group_indexes(index: dict, out: Path) -> None:
    """One small {"<id>": "<hash>"} file per group, loaded lazily by the frontend."""
    groups: dict[str, dict[str, str]] = {}
    for key, meta in sorted(index.items()):
        if meta.get("hash"):
            group, base = key.rsplit("/", 1)
            groups.setdefault(group, {})[base] = meta["hash"]
    out.mkdir(parents=True, exist_ok=True)
    for old in out.glob("*.json"):
        old.unlink()
    for group, names in groups.items():
        ordered = dict(sorted(names.items(), key=lambda kv: (len(kv[0]), kv[0])))
        (out / f"{group.replace('/', '-')}.json").write_text(json.dumps(ordered, separators=(",", ":")), encoding="utf-8")


def remove_files(out: Path, key: str) -> None:
    group, base = key.rsplit("/", 1)
    for old in (out / group).glob(f"{glob.escape(base)}.*.webp"):
        if old.name.count(".") == 2:  # <id>.<hash>.webp of an earlier export
            old.unlink()


def installed_manifest(game: Path, package: str, parse, resolve) -> dict:
    """The client's selected manifest of a package (hot update cache first, then the built-in one)."""
    cached = sorted((game / "AzurPromilia_Data/.res" / package / "ManifestFiles").glob(f"*_{package}_*.bytes"))
    builtin = sorted((game / "AzurPromilia_Data/StreamingAssets/.res" / package).glob(f"*_{package}_*.bytes"))
    candidates = cached or builtin
    if len(candidates) != 1:
        raise SystemExit(f"{package}: expected one installed manifest, got {candidates}")
    path = candidates[0]
    expected = path.with_suffix(".hash").read_text().strip().lower()
    if hashlib.md5(path.read_bytes()).hexdigest() != expected:
        raise SystemExit(f"{package}: manifest hash mismatch ({path})")
    manifest = parse(path)
    resolve(manifest, game)
    return manifest


def vstring_pool(base: dict, game: Path, read_bundle, decrypt_mmap, Strings):
    """The VString pool of raw_base_package (asset paths of every manifest are VString keys)."""
    found = []
    for bundle in base["bundles"]:
        pack = base["packs"][bundle["pack_id"]]
        if not pack["local_paths"]:
            continue
        with open(pack["local_paths"][0], "rb") as f:
            f.seek(bundle["file_offset"])
            if f.read(8) != struct.pack("<II", 1, 4112):  # MMap header
                continue
        raw, _ = read_bundle(base, bundle, game)
        data, origin = decrypt_mmap(raw)
        try:
            strings = Strings(data, origin)
            first = strings.get(1)
        except (ValueError, IndexError, struct.error, UnicodeError):
            continue
        if isinstance(first, str) and "vstring" in first.lower():
            found.append(strings)
    if len(found) != 1:
        raise SystemExit(f"VString pool is not unique in raw_base_package: {len(found)}")
    return found[0]


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--game", type=Path, required=True, help="game install dir (read only)")
    ap.add_argument("--tools", type=Path, required=True, help="Azur Promilia analysis project tools dir (export_resources.py etc.)")
    ap.add_argument("--res", type=Path, required=True, help="server res dir (Excel/*.json)")
    ap.add_argument("--out", type=Path, default=FRONTEND / "public/games/azurpromilia", help="image dir (served as static files)")
    ap.add_argument("--index", type=Path, default=FRONTEND / "src/games/azurpromilia/assets", help="per-group id -> hash indexes")
    ap.add_argument("--state", type=Path, default=Path(__file__).with_name("state.json"), help="incremental export state")
    ap.add_argument("--only", default="", help="comma separated groups, e.g. heroes,items/common_item,covers")
    ap.add_argument("--lang", default="cht", help="client language of localized assets (ResourcesLang/<lang>)")
    ap.add_argument("--size", type=int, default=MAX_SIZE, help="max width / height of an icon")
    ap.add_argument("--quality", type=int, default=82)
    ap.add_argument("--force", action="store_true", help="re-export even if the source bundle is unchanged")
    ap.add_argument("--jobs", type=int, default=os.cpu_count() or 4, help="parallel exports")
    a = ap.parse_args()

    sys.path.insert(0, str(a.tools.resolve()))
    from parse_manifest import parse  # noqa: E402  (Azur Promilia analysis tools)
    from export_resources import read_bundle, resolve, unity_env  # noqa: E402
    from export_server_resources import Strings, decrypt_mmap  # noqa: E402
    from PIL import Image  # noqa: E402

    only = {g.strip().strip("/") for g in a.only.split(",") if g.strip()}
    wanted = {k: v for k, v in wanted_images(a.res).items() if not only or group_of(k) in only}
    with_app = not only or group_of(APP_ICON) in only

    def fit_of(group: str) -> tuple[int, int | None, bool]:
        return FITS.get(group, (a.size, a.size, True))

    base = installed_manifest(a.game, "raw_base_package", parse, resolve)
    strings = vstring_pool(base, a.game, read_bundle, decrypt_mmap, Strings)
    manifest = installed_manifest(a.game, "default_package", parse, resolve)
    roots = ("assets/arts/ui/", "assets/resourcesassets/ui/", f"assets/resourceslang/{a.lang.lower()}/ui/")
    assets: dict[str, tuple[str, dict]] = {}  # lower-case path -> (path, manifest asset)
    for asset in manifest["assets"]:
        path = strings.get(asset["path_key"])
        if path and path.lower().startswith(roots):
            assets[path.lower()] = (path, asset)

    state: dict = json.loads(a.state.read_text(encoding="utf-8")) if a.state.exists() else {}
    stats = Counter()
    no_image: dict[str, list[str]] = defaultdict(list)
    not_in_client: dict[str, list[str]] = defaultdict(list)
    jobs: dict[tuple[str, tuple], list[tuple[str, str]]] = defaultdict(list)  # (asset, fit) -> [(key, src)]
    keep: set[str] = set()
    for key, src in sorted(wanted.items()):
        group, name = key.rsplit("/", 1)
        if not src:
            no_image[group].append(name)
            continue
        found = next((assets[c.lower()] for c in asset_candidates(src, a.lang) if c.lower() in assets), None)
        if found is None:
            not_in_client[group].append(f"{name} ({src})")
            continue
        path, asset = found
        keep.add(key)
        max_w, max_h, full = fit_of(group)
        fit_id = f"{max_w}x{max_h or ''}{'' if full else ' trim'}"
        crc = f"{manifest['bundles'][asset['bundle_id']]['crc']:08x}"
        prev = state.get(key, {})
        dst = a.out / f"{key}.{prev.get('hash', '')}.webp"
        same = (prev.get("src"), prev.get("asset"), prev.get("crc"), prev.get("fit")) == (src, path, crc, fit_id)
        if not a.force and same and prev.get("hash") and dst.exists():
            stats["same"] += 1
            continue
        jobs[(path, (max_w, max_h, full, fit_id))].append((key, src))

    if with_app:
        keep.add(APP_ICON)

    # keys no longer produced (row removed, image cleared or missing): drop files and state
    for key in [k for k in state if (not only or group_of(k) in only) and k not in keep]:
        remove_files(a.out, key)
        del state[key]
        stats["removed"] += 1

    def export(path: str, spec: tuple) -> tuple[dict, bytes]:
        max_w, max_h, full, fit_id = spec
        _, asset = assets[path.lower()]
        bundle = manifest["bundles"][asset["bundle_id"]]
        env = unity_env(manifest, bundle, a.game)
        container = []
        for obj in env.objects:
            if obj.type.name == "AssetBundle":
                container = [v["asset"]["m_PathID"] for p, v in obj.read_typetree()["m_Container"] if p.lower() == path.lower()]
        objs = {o.path_id: o for o in env.objects}
        mine = [objs[p] for p in container if p in objs]
        sprite = next((o for o in mine if o.type.name == "Sprite"), None)
        texture = next((o for o in mine if o.type.name == "Texture2D"), None)
        if sprite is None and texture is None:
            raise ValueError("no Sprite / Texture2D for the asset path")
        img = sprite_image(sprite.read(), Image, full) if sprite else texture.read().image.convert("RGBA")
        w, h = fit(img.width, img.height, max_w, max_h)
        if (w, h) != img.size:
            img = img.resize((w, h), Image.LANCZOS)
        buf = io.BytesIO()
        img.save(buf, "WEBP", quality=a.quality, method=6)
        body = buf.getvalue()
        return {"w": w, "h": h, "asset": path, "crc": f"{bundle['crc']:08x}", "fit": fit_id, "hash": hashlib.sha256(body).hexdigest()[:8]}, body

    with ThreadPoolExecutor(max(1, a.jobs)) as pool:
        futures = {pool.submit(export, path, spec): (path, spec) for path, spec in jobs}
        for f in as_completed(futures):
            job = futures[f]
            try:
                meta, body = f.result()
            except Exception as e:  # keep going; report at the end
                stats["error"] += len(jobs[job])
                print("error", job[0], e)
                continue
            for key, src in jobs[job]:
                group, base_name = key.rsplit("/", 1)
                remove_files(a.out, key)
                (a.out / group).mkdir(parents=True, exist_ok=True)
                (a.out / group / f"{base_name}.{meta['hash']}.webp").write_bytes(body)
                state[key] = {"w": meta["w"], "h": meta["h"], "src": src, **{k: meta[k] for k in ("asset", "crc", "fit", "hash")}}
                stats["ok"] += 1

    if with_app:
        raw = exe_icon(a.game / APP_EXE)
        crc = hashlib.sha256(raw).hexdigest()[:8]
        fit_id = f"{APP_ICON_SIZE}x{APP_ICON_SIZE}"
        prev = state.get(APP_ICON, {})
        same = (prev.get("src"), prev.get("crc"), prev.get("fit")) == (APP_EXE, crc, fit_id)
        if not a.force and same and prev.get("hash") and (a.out / f"{APP_ICON}.{prev['hash']}.webp").exists():
            stats["same"] += 1
        else:
            img = Image.open(io.BytesIO(raw)).convert("RGBA")
            w, h = fit(img.width, img.height, APP_ICON_SIZE, APP_ICON_SIZE)
            if (w, h) != img.size:
                img = img.resize((w, h), Image.LANCZOS)
            buf = io.BytesIO()
            img.save(buf, "WEBP", quality=a.quality, method=6)
            body = buf.getvalue()
            digest = hashlib.sha256(body).hexdigest()[:8]
            remove_files(a.out, APP_ICON)
            (a.out / group_of(APP_ICON)).mkdir(parents=True, exist_ok=True)
            (a.out / f"{APP_ICON}.{digest}.webp").write_bytes(body)
            state[APP_ICON] = {"w": w, "h": h, "src": APP_EXE, "asset": "RT_GROUP_ICON", "crc": crc, "fit": fit_id, "hash": digest}
            stats["ok"] += 1

    a.state.parent.mkdir(parents=True, exist_ok=True)
    a.state.write_text(json.dumps(dict(sorted(state.items())), ensure_ascii=False, indent=1) + "\n", encoding="utf-8")
    write_group_indexes(state, a.index)

    print("groups:")
    for group in sorted({group_of(k) for k in state}):
        files = list((a.out / group).glob("*.webp")) if (a.out / group).exists() else []
        print(f"  {group}: {len(files)} files, {sum(p.stat().st_size for p in files) / 1e6:.2f} MB")
    for title, rows in (("rows without an image", no_image), ("image not in default_package", not_in_client)):
        for group, ids in sorted(rows.items()):
            print(f"{title} · {group} ({len(ids)}): {', '.join(ids)}")
    total = sum(p.stat().st_size for p in a.out.rglob("*.webp")) if a.out.exists() else 0
    files = sum(1 for _ in a.out.rglob("*.webp")) if a.out.exists() else 0
    print(f"done: {dict(stats)} · {files} files · {total / 1e6:.1f} MB · default_package {manifest['header']['version']}")
    return 1 if stats["error"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
