#!/usr/bin/env python3
"""
根据 assets/logo/hxb.png 生成网站常用图标。

输出目录：assets/site-icons/
生成文件：
- favicon-16x16.png
- favicon-32x32.png
- apple-touch-icon.png (180x180)
- android-chrome-192x192.png
- android-chrome-512x512.png
- mstile-150x150.png
- favicon.ico (16/32/48)
- site.webmanifest
"""

from __future__ import annotations

import json
from pathlib import Path

try:
    from PIL import Image
except Exception as exc:  # pragma: no cover
    raise SystemExit(
        "[ERROR] 需要 Pillow 才能生成图标。\n"
        "请先安装：python3 -m pip install Pillow\n"
        f"详细错误：{exc}"
    )


ROOT = Path(__file__).resolve().parent
SOURCE = ROOT / "logo" / "hxb.png"
OUTPUT = ROOT / "site-icons"


def square_crop(img: Image.Image) -> Image.Image:
    w, h = img.size
    side = min(w, h)
    left = (w - side) // 2
    top = (h - side) // 2
    return img.crop((left, top, left + side, top + side))


def save_png(base: Image.Image, size: int, filename: str) -> None:
    icon = base.resize((size, size), Image.Resampling.LANCZOS)
    icon.save(OUTPUT / filename, format="PNG", optimize=True)


def main() -> None:
    if not SOURCE.exists():
        raise SystemExit(f"[ERROR] 未找到源图：{SOURCE}")

    OUTPUT.mkdir(parents=True, exist_ok=True)

    src = Image.open(SOURCE).convert("RGBA")
    base = square_crop(src)

    save_png(base, 16, "favicon-16x16.png")
    save_png(base, 32, "favicon-32x32.png")
    save_png(base, 48, "favicon-48x48.png")
    save_png(base, 150, "mstile-150x150.png")
    save_png(base, 180, "apple-touch-icon.png")
    save_png(base, 192, "android-chrome-192x192.png")
    save_png(base, 512, "android-chrome-512x512.png")

    # favicon.ico（多尺寸）
    ico_sizes = [(16, 16), (32, 32), (48, 48)]
    ico = base.resize((48, 48), Image.Resampling.LANCZOS)
    ico.save(OUTPUT / "favicon.ico", format="ICO", sizes=ico_sizes)

    manifest = {
        "name": "StuHelper",
        "short_name": "StuHelper",
        "icons": [
            {"src": "/assets/site-icons/android-chrome-192x192.png", "sizes": "192x192", "type": "image/png"},
            {"src": "/assets/site-icons/android-chrome-512x512.png", "sizes": "512x512", "type": "image/png"},
        ],
        "theme_color": "#ffffff",
        "background_color": "#ffffff",
        "display": "standalone",
    }
    (OUTPUT / "site.webmanifest").write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )

    print(f"[OK] 图标已生成到: {OUTPUT}")


if __name__ == "__main__":
    main()
