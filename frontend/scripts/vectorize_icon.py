"""Vectorize dual-tone UI PNG icons into two-path SVG (marching squares).

Usage: python frontend/scripts/vectorize_icon.py frontend/design/ui-source/icons/<icon>.png [--json]
Layers follow docs/02-需求与设计/UI设计细节.md:
  dark  = alpha > 150
  light = 20 < alpha <= 150
"""
import json
import sys
from pathlib import Path

from PIL import Image

DARK_ALPHA = 150
LIGHT_ALPHA = 20


def load_masks(path):
    img = Image.open(path).convert("RGBA")
    w, h = img.size
    alpha = [px[3] for px in img.getdata()]
    dark = [[alpha[y * w + x] > DARK_ALPHA for x in range(w)] for y in range(h)]
    light = [[LIGHT_ALPHA < alpha[y * w + x] <= DARK_ALPHA for x in range(w)] for y in range(h)]
    return w, h, dark, light


def marching_squares(mask, w, h):
    # pad so shapes touching the border still close
    grid = [[False] * (w + 2) for _ in range(h + 2)]
    for y in range(h):
        row = grid[y + 1]
        src = mask[y]
        for x in range(w):
            row[x + 1] = src[x]

    segments = []
    for cy in range(h + 1):
        for cx in range(w + 1):
            tl = grid[cy][cx]
            tr = grid[cy][cx + 1]
            br = grid[cy + 1][cx + 1]
            bl = grid[cy + 1][cx]
            idx = (tl << 3) | (tr << 2) | (br << 1) | bl
            if idx in (0, 15):
                continue
            # sample coords: original pixel (x,y) center => (x, y)
            x0, y0 = cx - 1, cy - 1
            top = (x0 + 0.5, y0)
            right = (x0 + 1, y0 + 0.5)
            bottom = (x0 + 0.5, y0 + 1)
            left = (x0, y0 + 0.5)
            table = {
                1: [(left, bottom)],
                2: [(bottom, right)],
                3: [(left, right)],
                4: [(right, top)],
                5: [(left, top), (bottom, right)],
                6: [(bottom, top)],
                7: [(left, top)],
                8: [(top, left)],
                9: [(top, bottom)],
                10: [(left, bottom), (top, right)],
                11: [(top, right)],
                12: [(right, left)],
                13: [(right, bottom)],
                14: [(bottom, left)],
                15: [],
            }
            segments.extend(table[idx])
    return segments


def chain(segments):
    key = lambda p: (round(p[0] * 2), round(p[1] * 2))
    adj = {}
    for i, (a, b) in enumerate(segments):
        adj.setdefault(key(a), []).append(i)
        adj.setdefault(key(b), []).append(i)
    used = [False] * len(segments)
    loops = []
    for i, (a, b) in enumerate(segments):
        if used[i]:
            continue
        used[i] = True
        loop = [a, b]
        cur = b
        while True:
            nxt = None
            for j in adj.get(key(cur), []):
                if not used[j]:
                    nxt = j
                    break
            if nxt is None:
                break
            used[nxt] = True
            p, q = segments[nxt]
            cand = q if key(p) == key(cur) else p
            loop.append(cand)
            cur = cand
        loops.append(loop)
    return loops


def fmt(v):
    return f"{v:.2f}".rstrip("0").rstrip(".")


def path_d(loops):
    parts = []
    for loop in loops:
        if len(loop) < 3:
            continue
        pts = loop
        # drop last point if it repeats the first
        if key_eq(pts[0], pts[-1]):
            pts = pts[:-1]
        if len(pts) < 3:
            continue
        d = [f"M{fmt(pts[0][0])} {fmt(pts[0][1])}"]
        for p in pts[1:]:
            d.append(f"L{fmt(p[0])} {fmt(p[1])}")
        d.append("Z")
        parts.append(" ".join(d))
    return " ".join(parts)


def key_eq(a, b):
    return abs(a[0] - b[0]) < 1e-6 and abs(a[1] - b[1]) < 1e-6


def main():
    path = Path(sys.argv[1])
    w, h, dark, light = load_masks(path)
    out = {}
    for name, mask in (("dark", dark), ("light", light)):
        loops = chain(marching_squares(mask, w, h))
        out[name] = {"loops": len(loops), "d": path_d(loops)}
    if "--json" in sys.argv:
        print(json.dumps({"size": [w, h], **out}, ensure_ascii=False))
    else:
        print(f"{path.name}: {w}x{h}")
        for name in ("light", "dark"):
            print(f"  {name}: {out[name]['loops']} loops, {len(out[name]['d'])} chars")
            print(out[name]["d"][:200] + "...")
            print()


if __name__ == "__main__":
    main()
