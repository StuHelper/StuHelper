# `/assets`

站点静态资源目录（logo、图标等）。

## 图标生成

源图：`assets/logo/hxb.png`

生成脚本：`assets/generate_icons.py`

```bash
python3 -m pip install Pillow
python3 assets/generate_icons.py
```

输出目录：`assets/site-icons/`

- `favicon-16x16.png`
- `favicon-32x32.png`
- `favicon.ico`
- `apple-touch-icon.png`
- `android-chrome-192x192.png`
- `android-chrome-512x512.png`
- `mstile-150x150.png`
- `site.webmanifest`
