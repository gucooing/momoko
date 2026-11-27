"""导出蓝色星原的养成范围（前端资源，GM 修改角色 / 奇波时把超出范围的值夹到最大或最小）。

从游戏服务端的资源目录（server/res 的 Excel 导出 JSON）读表，规则与游戏服务端 gdconf 一致，写到
frontend/src/games/azurpromilia/growth.json：
  heroLevel / petLevel / favorLevel   [最小, 最大]：hero_level、pet_level 的等级，hero_favorability_exp 的好感等级
  heroes   {英雄 ID: [最低突破, [升到最低突破 + i 需要的等级…], 最低星级, 最高星级]}
           突破需要的等级：从最低突破逐级升上去，每一级要求等级不低于该级的 rankLevelLimit（gdconf.HeroRankLevelNeed）；
           星级：hero_grade 里的最低星级起，连续存在、不超过 HERO_MAX_GRADE 与 gradeUpItem 个数（heroGradeValid）
  starsoul {星魂模板 ID: [[各阶节点数…], [按推进顺序每个状态需要的等级…]]}
           星魂从第 1 阶第 1 节点逐个推进，最后一阶只能到第 1 节点；到达一个状态需要等级不低于之前每个节点的
           unlocklevel（gdconf.PetStarsoulLevelNeed）
  pets     {奇波 ID: 星魂模板 ID}（starsoulId 为 0 的奇波没有星魂，不列出）

用法：python -I -B frontend/scripts/games/azurpromilia/export-growth.py --res E:/ps/AzurPromilia/server/res
"""

import argparse
import json
import os


def read_rows(res, table):
    with open(os.path.join(res, "Excel", table + ".json"), encoding="utf-8") as f:
        data = json.load(f)
    return data["list"] if isinstance(data, dict) else data


def level_range(rows, field):
    levels = [r[field] for r in rows]
    return [min(levels), max(levels)]


def game_const(res, title):
    for r in read_rows(res, "game"):
        if r.get("title") == title:
            return int(r["value"])
    raise SystemExit(f"game.json has no {title}")


def heroes(res):
    max_grade = game_const(res, "HERO_MAX_GRADE")
    ranks = {}
    for r in read_rows(res, "talent_rank"):
        ranks.setdefault(r["heroId"], {})[r["rank"]] = r["rankLevelLimit"]
    grades = {}
    for r in read_rows(res, "hero_grade"):
        grades.setdefault(r["heroId"], set()).add(r["grade"])
    ups = {r["id"]: len(r.get("gradeUpItem") or []) for r in read_rows(res, "hero")}

    out = {}
    for hero, by_rank in sorted(ranks.items()):
        low = min(by_rank)
        needs, need = [0], 0
        for rank in range(low + 1, max(by_rank) + 1):
            need = max(need, by_rank[rank])
            needs.append(need)
        g = grades.get(hero, set())
        g_low = min(g) if g else 0
        g_high = g_low
        while g_high + 1 <= min(max_grade, ups.get(hero, 0)) and g_high + 1 in g:
            g_high += 1
        out[str(hero)] = [low, needs, g_low, g_high]
    return out


def starsouls(res):
    nodes = {}
    for r in read_rows(res, "pet_starsoul"):
        nodes.setdefault(r["templateId"], {}).setdefault(r["group"], {})[r["node"]] = r["unlocklevel"]
    out = {}
    for tpl, stages in sorted(nodes.items()):
        counts = [len(stages[s]) for s in range(1, len(stages) + 1)]
        order = [(s, n) for s in range(1, len(stages)) for n in range(1, counts[s - 1] + 1)] + [(len(stages), 1)]
        needs, need = [], 0
        for s, n in order:
            needs.append(need)
            need = max(need, stages[s][n])
        out[str(tpl)] = [counts, needs]
    return out


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--res", required=True, help="游戏服务端 res 目录")
    here = os.path.dirname(os.path.abspath(__file__))
    ap.add_argument("--out", default=os.path.join(here, "..", "..", "..", "src", "games", "azurpromilia", "growth.json"))
    a = ap.parse_args()

    souls = starsouls(a.res)
    data = {
        "heroLevel": level_range(read_rows(a.res, "hero_level"), "lv"),
        "petLevel": level_range(read_rows(a.res, "pet_level"), "lv"),
        "favorLevel": level_range(read_rows(a.res, "hero_favorability_exp"), "favorabilityLevel"),
        "heroes": heroes(a.res),
        "starsoul": souls,
        "pets": {str(r["id"]): r["starsoulId"] for r in read_rows(a.res, "pet") if str(r.get("starsoulId", 0)) in souls},
    }
    with open(os.path.normpath(a.out), "w", encoding="utf-8", newline="\n") as f:
        json.dump(data, f, ensure_ascii=False, separators=(",", ":"))
        f.write("\n")
    print(f"heroes {len(data['heroes'])} · starsoul templates {len(souls)} · pets {len(data['pets'])} · {os.path.getsize(a.out)} bytes")


if __name__ == "__main__":
    main()
