"""导出蓝色星原的名称表（前端资源）。

从游戏服务端的资源目录（蓝色星原服务端 server/res：Excel 导出 JSON 与 Localization）读取奖励种类与各种类的名称，写到
frontend/src/games/azurpromilia：
  names/<lang>/<kind>.json   名称表（id → 名称），lang 为 zh（游戏的繁体中文）与 en；kind 为奖励种类的表名
  names/<lang>/reward_type.json   奖励种类的名称（种类数值 → 名称）
  reward-types.json   有名称表的奖励种类：[{"type": 种类数值, "kind": 表名}]

用法：python -I -B frontend/scripts/games/azurpromilia/export-names.py --res E:/ps/AzurPromilia/server/res
"""

import argparse
import json
import os

LANGS = {"zh": "cht", "en": "en"}


def read_rows(path):
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    return data["list"] if isinstance(data, dict) else data


def localization(res, lang, group):
    path = os.path.join(res, "Localization", lang, group.removeprefix("lang_") + ".json")
    if not os.path.exists(path):
        return {}
    return {r["id"]: r.get("value", "") for r in read_rows(path)}


def names_of(res, rows, field, lang):
    out = {}
    tables = {}
    for r in rows:
        ref = r.get(field)
        if "id" not in r or not isinstance(ref, dict) or not ref.get("group"):
            continue
        group = ref["group"]
        if group not in tables:
            tables[group] = localization(res, lang, group)
        value = tables[group].get(ref.get("langKey"), "")
        if value:
            out[str(r["id"])] = value
    return out


def write(path, data):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        json.dump(data, f, ensure_ascii=False, separators=(",", ":"), sort_keys=True)
        f.write("\n")


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    parser = argparse.ArgumentParser()
    parser.add_argument("--res", required=True, help="蓝色星原服务端的 res 目录")
    parser.add_argument("--out", default=os.path.normpath(os.path.join(here, "../../../src/games/azurpromilia")))
    args = parser.parse_args()

    reward_types = read_rows(os.path.join(args.res, "Excel", "reward_type.json"))
    kinds = []
    for t in sorted(reward_types, key=lambda r: r["id"]):
        table = t.get("tableName", "")
        path = os.path.join(args.res, "Excel", table + ".json")
        if not table or not os.path.exists(path):
            continue
        rows = read_rows(path)
        if not rows or "id" not in rows[0] or not isinstance(rows[0].get("name"), dict):
            continue
        for lang, source in LANGS.items():
            write(os.path.join(args.out, "names", lang, table + ".json"), names_of(args.res, rows, "name", source))
        kinds.append({"type": t["id"], "kind": table})
    for lang, source in LANGS.items():
        write(os.path.join(args.out, "names", lang, "reward_type.json"), names_of(args.res, reward_types, "desc", source))
    write(os.path.join(args.out, "reward-types.json"), kinds)
    print("exported %d reward kinds to %s" % (len(kinds), args.out))


if __name__ == "__main__":
    main()
