"""Export localized 学マス names and texts (characters, items, cards, costumes, music, produce, towers,
skill cards, P items, memory abilities) for the frontend.

Like the images, names are frontend resources: the game server (MGP) returns IDs and the original
(Japanese) names; the frontend shows the localized name when the current language has one and
falls back to the server's name otherwise. Run this offline after a game / translation update and
commit the output.

Sources:
  --localization  the 学マス translation package (masterTrans/*.json + localization.json), Simplified Chinese
  --master        the server's master data (server/res/Excel/*.json): English character names
                  (alphabetFirstName / alphabetLastName) and IdolCardSkin -> IdolCard for unnamed skins

Output (one small file per kind and language, loaded lazily by src/games/names.ts):
  src/games/gakumas/names/<lang>/<kind>.json     {"<id>": "<name>"}; kinds match the MGP catalog kinds
  besides the catalog kinds, texts use their own kinds and keys:
    produceCard         "<id>" (not upgraded) and "<id>+<n>" (upgraded n times; the name carries the "+")
    produceItemText     "<id>": the P item effect text
    memoryAbilityText   "<id>@<level>": the effect text of a memory ability level (its ProduceSkill)
  texts are the description segments joined without rich text tags (<nobr>, <color=...>)
  lang = zh (translation package), en (characters and currencies only), ja (the game's original names
  from the master data: the fallback when the current language has no translation, so the UI never
  has to show an ID)

Only real translations are written; there is no English source for items, cards, costumes or music,
so in English those show the original names.

Usage (Windows example):
  python frontend/scripts/games/gakumas/export-names.py ^
      --localization C:/Users/gucooing/Desktop/gakumas/localization ^
      --master       C:/Users/gucooing/Desktop/gakumas/server/res/Excel
"""
from __future__ import annotations

import argparse
import json
import re
from pathlib import Path

FRONTEND = Path(__file__).resolve().parents[3]


def rows(path: Path) -> list[dict]:
    d = json.loads(path.read_text(encoding="utf-8"))
    if isinstance(d, list):
        return d
    if "data" in d:
        return d["data"]
    return next(iter(d.values()))


TAG = re.compile(r"</?[a-zA-Z][^<>]*>")


def description(r: dict) -> str:
    """The plain text of a row's produceDescriptions."""
    return TAG.sub("", "".join((d.get("text") or "") for d in r.get("produceDescriptions") or [])).strip()


def produce_cards(path: Path) -> dict[str, str]:
    out: dict[str, str] = {}
    for r in rows(path):
        name, n = (r.get("name") or "").strip(), r.get("upgradeCount") or 0
        if r.get("id") and name:
            out.setdefault(r["id"] if not n else f"{r['id']}+{n}", name)
    return out


def item_texts(path: Path) -> dict[str, str]:
    out: dict[str, str] = {}
    for r in rows(path):
        text = description(r)
        if r.get("id") and text:
            out.setdefault(r["id"], text)
    return out


def ability_texts(abilities: Path, skills: Path) -> dict[str, str]:
    """Memory ability (id, level) -> the text of its skill (skillId, level)."""
    texts: dict[tuple[str, int], str] = {}
    for r in rows(skills):
        text = description(r)
        if r.get("id") and text:
            texts.setdefault((r["id"], r.get("level") or 0), text)
    out: dict[str, str] = {}
    for r in rows(abilities):
        text = texts.get((r.get("skillId") or "", r.get("level") or 0))
        if r.get("id") and text:
            out.setdefault(f"{r['id']}@{r.get('level') or 0}", text)
    return out


def by_id(path: Path, field: str) -> dict[str, str]:
    out: dict[str, str] = {}
    for r in rows(path):
        v = (r.get(field) or "").strip()
        if r.get("id") and v and r["id"] not in out:
            out[r["id"]] = v
    return out


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--localization", type=Path, required=True)
    ap.add_argument("--master", type=Path, required=True)
    ap.add_argument("--out", type=Path, default=FRONTEND / "src/games/gakumas/names")
    a = ap.parse_args()

    tr = a.localization / "masterTrans"
    zh: dict[str, dict[str, str]] = {}

    # 角色：姓 + 名
    zh["character"] = {}
    for r in rows(tr / "Character.json"):
        name = (r.get("lastName") or "") + (r.get("firstName") or "")
        if r.get("id") and name.strip():
            zh["character"][r["id"]] = name.strip()

    zh["item"] = by_id(tr / "Item.json", "name")
    idol_cards = by_id(tr / "IdolCard.json", "name")
    zh["idolCard"] = idol_cards
    # 偶像卡皮肤：皮肤自己的名字，没有时用所属偶像卡的名字（与服务端目录的规则相同）
    skin_names = by_id(tr / "IdolCardSkin.json", "name")
    zh["idolCardSkin"] = {}
    for r in rows(a.master / "IdolCardSkin.json"):
        sid, cid = r.get("id"), r.get("idolCardId")
        name = skin_names.get(sid) or idol_cards.get(cid or "")
        if sid and name:
            zh["idolCardSkin"][sid] = name
    zh["supportCard"] = by_id(tr / "SupportCard.json", "name")
    zh["costume"] = by_id(tr / "Costume.json", "name")
    zh["costumeHead"] = by_id(tr / "CostumeHead.json", "name")
    zh["music"] = by_id(tr / "Music.json", "title")
    zh["produceGroup"] = by_id(tr / "ProduceGroup.json", "name")
    zh["produce"] = by_id(tr / "Produce.json", "name")
    zh["tower"] = by_id(tr / "Tower.json", "title")
    zh["gashaButton"] = by_id(tr / "GashaButton.json", "name")
    zh["produceCard"] = produce_cards(tr / "ProduceCard.json")
    zh["produceItem"] = by_id(tr / "ProduceItem.json", "name")
    zh["produceItemText"] = item_texts(tr / "ProduceItem.json")
    # 记忆能力：主数据的 MemoryAbility（id、level → skillId）对上译文的 ProduceSkill
    zh["memoryAbilityText"] = ability_texts(a.master / "MemoryAbility.json", tr / "ProduceSkill.json")

    # 货币：MGP 的 currency 目录是固定的四项，名字取游戏界面文案
    ui = json.loads((a.localization / "localization.json").read_text(encoding="utf-8"))
    currency_keys = {
        "jewelFree": "common.jewel_free_only",
        "jewelPaid": "common.jewel_paid_only",
        "actionPoint": "common.ap",
        "exp": "common.user_exp",
    }
    zh["currency"] = {cid: ui[key] for cid, key in currency_keys.items() if ui.get(key)}

    en: dict[str, dict[str, str]] = {}
    # 角色英文名：官方罗马字（名 + 姓），转成首字母大写
    en["character"] = {}
    for r in rows(a.master / "Character.json"):
        first, last = (r.get("alphabetFirstName") or "").strip(), (r.get("alphabetLastName") or "").strip()
        if r.get("id") and (first or last):
            en["character"][r["id"]] = " ".join(p.title() for p in (first, last) if p)
    # 货币：界面用语
    en["currency"] = {"jewelFree": "Free jewels", "jewelPaid": "Paid jewels", "actionPoint": "AP", "exp": "Producer EXP"}

    # 原文名（日文，主数据）：当前语言没有译名时的兜底
    m = a.master
    ja: dict[str, dict[str, str]] = {}
    ja["character"] = {}
    for r in rows(m / "Character.json"):
        name = ((r.get("lastName") or "") + (r.get("firstName") or "")).strip()
        if r.get("id") and name:
            ja["character"][r["id"]] = name
    ja["item"] = by_id(m / "Item.json", "name")
    ja_cards = by_id(m / "IdolCard.json", "name")
    ja["idolCard"] = ja_cards
    ja["idolCardSkin"] = {}
    for r in rows(m / "IdolCardSkin.json"):
        sid = r.get("id")
        name = (r.get("name") or "").strip() or ja_cards.get(r.get("idolCardId") or "", "")
        if sid and name:
            ja["idolCardSkin"][sid] = name
    ja["supportCard"] = by_id(m / "SupportCard.json", "name")
    ja["costume"] = by_id(m / "Costume.json", "name")
    ja["costumeHead"] = by_id(m / "CostumeHead.json", "name")
    ja["music"] = by_id(m / "Music.json", "title")
    ja["produceGroup"] = by_id(m / "ProduceGroup.json", "name")
    ja["produce"] = by_id(m / "Produce.json", "name")
    ja["tower"] = by_id(m / "Tower.json", "title")
    ja["gashaButton"] = by_id(m / "GashaButton.json", "name")
    ja["produceCard"] = produce_cards(m / "ProduceCard.json")
    ja["produceItem"] = by_id(m / "ProduceItem.json", "name")
    ja["produceItemText"] = item_texts(m / "ProduceItem.json")
    ja["memoryAbilityText"] = ability_texts(m / "MemoryAbility.json", m / "ProduceSkill.json")

    for lang, kinds in (("zh", zh), ("en", en), ("ja", ja)):
        folder = a.out / lang
        folder.mkdir(parents=True, exist_ok=True)
        for old in folder.glob("*.json"):
            old.unlink()
        for kind, names in kinds.items():
            if names:
                (folder / f"{kind}.json").write_text(
                    json.dumps(dict(sorted(names.items())), ensure_ascii=False, separators=(",", ":")), encoding="utf-8"
                )
        print(lang, {k: len(v) for k, v in kinds.items()})
    version = (a.localization / "version.txt")
    if version.exists():
        print("translation package", version.read_text(encoding="utf-8").strip())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
