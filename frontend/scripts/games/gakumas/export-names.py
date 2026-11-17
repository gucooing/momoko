"""Export localized 学マス names and texts (characters, items, cards, costumes, music, produce, towers,
skill cards, P items, memory abilities) for the frontend.

Like the images, names are frontend resources: the game server (MGP) returns IDs and the original
(Japanese) names; the frontend shows the localized name when the current language has one and
falls back to the server's name otherwise. Run this offline after a game / translation update and
commit the output.

Sources:
  --localization  the 学マス translation package (masterTrans/*.json, localization.json and the generic
                  text tables), Simplified Chinese
  --master        the server's master data (server/res/Excel/*.json): original names, English character
                  names (alphabetFirstName / alphabetLastName), IdolCardSkin -> IdolCard, catalog fields
  --data          the server's hand-written data (server/data): seminar groups and exams, tower layers

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

Catalogs (src/games/gakumas/catalogs/<kind>.json, one array per catalog kind): the entries the pickers,
search and filters list, with the fields they filter and show (resourceType, rarity, type, characterId,
parentId, assetId, characterAsset, levels; empty fields omitted). Names are not repeated there: the
frontend takes them from the name tables. The entries follow the game server's data: items, idol card
skins (rarity, plan type and character of their idol card), support cards, playable characters,
costumes, hairstyles, music, produce groups and produces, skill cards (not upgraded), P items, memory
abilities (levels), seminar groups and exams (parentId = group, in group order), towers (type = layer
count) and the four currencies. Gashas are GM-edited configuration and are not exported.

Usage (Windows example):
  python frontend/scripts/games/gakumas/export-names.py ^
      --localization C:/Users/gucooing/Desktop/gakumas/localization ^
      --master       C:/Users/gucooing/Desktop/gakumas/server/res/Excel ^
      --data         C:/Users/gucooing/Desktop/gakumas/server/data
"""
from __future__ import annotations

import argparse
import collections
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


def generic_texts(localization: Path) -> dict[str, str]:
    """原文 -> 译文：generic.json 与 genericTrans/**/*.json（不含数字模板 .fmt.json 与分段条目），后读的覆盖先读的。"""
    out: dict[str, str] = {}
    files = [localization / "generic.json"] + sorted(
        p for p in (localization / "genericTrans").rglob("*.json") if not p.name.endswith(".fmt.json")
    )
    for path in files:
        if not path.is_file():
            continue
        d = json.loads(path.read_text(encoding="utf-8-sig"))
        if isinstance(d, dict):
            out.update({k: v for k, v in d.items() if isinstance(v, str) and v and not k.startswith("[__split__]")})
    return out


# 货币：MGP 的四种货币与其奖励资源类型（ResourceType）；原文名与游戏服务端相同
CURRENCY_NAMES = {"jewelFree": "ジュエル（無償）", "jewelPaid": "ジュエル（有償）", "actionPoint": "AP", "exp": "プロデューサーEXP"}
CURRENCY_TYPES = {"jewelFree": 1100, "jewelPaid": 1101, "actionPoint": 8, "exp": 9}
# 可作为奖励的目录的资源类型（ResourceType）
RESOURCE = {"item": 2, "supportCard": 4, "idolCardSkin": 5, "music": 12, "costumeHead": 13, "costume": 14}


def entry(id: str, **fields) -> dict:
    """目录条目：ID 加非空字段。"""
    return {"id": id, **{k: v for k, v in fields.items() if v not in (None, "", 0, False, [])}}


def by_id_list(entries: list[dict]) -> list[dict]:
    seen: dict[str, dict] = {}
    for e in entries:
        seen.setdefault(e["id"], e)
    return sorted(seen.values(), key=lambda e: e["id"])


def write_catalogs(folder: Path, m: Path, seminar_groups: list[dict], tower_layers: list[dict]) -> None:
    """目录条目（名称取名称表），每类一个文件。"""
    cards = {r["id"]: r for r in rows(m / "IdolCard.json")}
    card = lambda r: cards.get(r.get("idolCardId") or "", {})
    groups = {pid: r["id"] for r in rows(m / "ProduceGroup.json") for pid in r.get("produceIds") or []}
    abilities: dict[str, dict] = {}
    for r in rows(m / "MemoryAbility.json"):
        v = abilities.setdefault(r["id"], {"rarity": r.get("rarity") or 0, "levels": set()})
        v["levels"].add(r.get("level") or 0)
    layers = collections.Counter(l["towerId"] for l in tower_layers)
    catalogs = {
        "currency": [entry(k, resourceType=v) for k, v in CURRENCY_TYPES.items()],
        "item": by_id_list([
            entry(r["id"], resourceType=RESOURCE["item"], rarity=r.get("rarity"), type=r.get("type"),
                  characterId=r.get("characterId"), assetId=r.get("assetId"))
            for r in rows(m / "Item.json")
        ]),
        "idolCardSkin": by_id_list([
            entry(r["id"], resourceType=RESOURCE["idolCardSkin"], rarity=card(r).get("rarity"), type=card(r).get("planType"),
                  characterId=card(r).get("characterId"), parentId=r.get("idolCardId"), assetId=r.get("assetId"))
            for r in rows(m / "IdolCardSkin.json")
        ]),
        "supportCard": by_id_list([
            entry(r["id"], resourceType=RESOURCE["supportCard"], rarity=r.get("rarity"), type=r.get("type"), assetId=r.get("assetId"))
            for r in rows(m / "SupportCard.json")
        ]),
        "character": by_id_list([entry(r["id"]) for r in rows(m / "Character.json") if r.get("isPlayable")]),
        "costume": by_id_list([
            entry(r["id"], resourceType=RESOURCE["costume"], characterId=r.get("characterId")) for r in rows(m / "Costume.json")
        ]),
        "costumeHead": by_id_list([
            entry(r["id"], resourceType=RESOURCE["costumeHead"], characterId=r.get("characterId")) for r in rows(m / "CostumeHead.json")
        ]),
        "music": by_id_list([
            entry(r["id"], resourceType=RESOURCE["music"], assetId=r.get("jacketAssetId")) for r in rows(m / "Music.json")
        ]),
        "produceGroup": by_id_list([entry(r["id"], type=r.get("type"), assetId=r.get("assetId")) for r in rows(m / "ProduceGroup.json")]),
        "produce": by_id_list([entry(r["id"], parentId=groups.get(r["id"]), assetId=r.get("assetId")) for r in rows(m / "Produce.json")]),
        "produceCard": by_id_list([
            entry(r["id"], rarity=r.get("rarity"), type=r.get("planType"), assetId=r.get("assetId"), characterAsset=r.get("isCharacterAsset"))
            for r in rows(m / "ProduceCard.json")
            if not r.get("upgradeCount")
        ]),
        "produceItem": by_id_list([entry(r["id"], rarity=r.get("rarity"), assetId=r.get("assetId")) for r in rows(m / "ProduceItem.json")]),
        "memoryAbility": [
            entry(i, rarity=v["rarity"], levels=sorted(v["levels"])) for i, v in sorted(abilities.items())
        ],
        "seminar": [
            e for g in seminar_groups for e in (entry(g["id"]), *(entry(x["id"], parentId=g["id"]) for x in g.get("exams", [])))
        ],
        "tower": [entry(t, type=n) for t, n in sorted(layers.items())],
    }
    folder.mkdir(parents=True, exist_ok=True)
    for old in folder.glob("*.json"):
        old.unlink()
    for kind, entries in catalogs.items():
        (folder / f"{kind}.json").write_text(json.dumps(entries, ensure_ascii=False, separators=(",", ":")), encoding="utf-8")
    print("catalogs", {k: len(v) for k, v in catalogs.items()})


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--localization", type=Path, required=True)
    ap.add_argument("--master", type=Path, required=True)
    ap.add_argument("--data", type=Path, required=True)
    ap.add_argument("--out", type=Path, default=FRONTEND / "src/games/gakumas/names")
    ap.add_argument("--catalogs", type=Path, default=FRONTEND / "src/games/gakumas/catalogs")
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

    # 研修：研修组与考试的名称在服务端手写数据里，译名取汉化包的 generic 文本
    texts = generic_texts(a.localization)
    groups = sorted(json.loads((a.data / "Seminar.json").read_text(encoding="utf-8"))["groups"], key=lambda g: g.get("order") or 0)
    ja["seminar"], zh["seminar"] = {}, {}
    for g in groups:
        for v in (g, *g.get("exams", [])):
            ja["seminar"][v["id"]] = v["name"]
            if texts.get(v["name"]):
                zh["seminar"][v["id"]] = texts[v["name"]]
    ja["currency"] = dict(CURRENCY_NAMES)

    write_catalogs(a.catalogs, m, groups, json.loads((a.data / "Tower.json").read_text(encoding="utf-8"))["layers"])

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
