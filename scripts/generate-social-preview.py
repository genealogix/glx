#!/usr/bin/env python3
"""Render .github/social-preview.png from public/logo.png + brand text.

The output is the 1280x640 image that a maintainer uploads via repo
Settings -> Social preview. GitHub does not expose an API for that
upload, so this script only produces the asset; the upload itself is
manual and must be re-done whenever the asset changes.

Usage:
    pip install Pillow numpy
    python scripts/generate-social-preview.py
"""

from __future__ import annotations

import sys
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw, ImageFont

REPO_ROOT = Path(__file__).resolve().parent.parent
LOGO_PATH = REPO_ROOT / "public" / "logo.png"
OUTPUT_PATH = REPO_ROOT / ".github" / "social-preview.png"

WIDTH, HEIGHT = 1280, 640
GRADIENT_START = (0x06, 0x57, 0xFF)  # logo gradient stop 0
GRADIENT_END = (0xE7, 0x91, 0xFF)    # logo gradient stop 1

LOGO_TARGET_HEIGHT = 380
LOGO_LEFT_PADDING = 80
TEXT_LEFT = 500

# The committed public/logo.png has a flat dark gray (#2F2F2F) backdrop
# baked in; we chroma-key it out so the tree silhouette sits cleanly on
# the gradient. Anything within BG_THRESHOLD of LOGO_BG_COLOR becomes
# fully transparent, with a soft ramp out to BG_RAMP for anti-aliased
# edges on the gradient strokes.
LOGO_BG_COLOR = (47, 47, 47)
BG_THRESHOLD = 12
BG_RAMP = 64

WORDMARK = "GENEALOGIX"
TAGLINE = "Evidence-first, Git-native genealogy data standard"
BULLETS = "YAML format  ·  Open specification  ·  Version controlled"

WORDMARK_SIZE = 96
TAGLINE_SIZE = 32
BULLETS_SIZE = 26

WORDMARK_GAP = 28
TAGLINE_GAP = 22

# Tried in order. First hit wins; missing all of them is a hard error
# because the PIL bitmap fallback renders ~10px and would ship a broken
# PNG without the maintainer noticing.
FONT_CANDIDATES_BOLD = [
    "arialbd.ttf",
    "Arial Bold.ttf",
    "DejaVuSans-Bold.ttf",
    "/System/Library/Fonts/Helvetica.ttc",
]
FONT_CANDIDATES_REGULAR = [
    "arial.ttf",
    "Arial.ttf",
    "DejaVuSans.ttf",
    "/System/Library/Fonts/Helvetica.ttc",
]


def load_font(candidates: list[str], size: int) -> ImageFont.FreeTypeFont:
    for name in candidates:
        try:
            return ImageFont.truetype(name, size=size)
        except OSError:
            continue
    raise RuntimeError(f"no usable font found in {candidates}")


def render_gradient() -> Image.Image:
    """Linear gradient projected onto the (WIDTH, HEIGHT) diagonal vector."""
    xs = np.arange(WIDTH)[None, :] * WIDTH
    ys = np.arange(HEIGHT)[:, None] * HEIGHT
    t = ((xs + ys) / (WIDTH * WIDTH + HEIGHT * HEIGHT))[..., None]
    start = np.array(GRADIENT_START)
    end = np.array(GRADIENT_END)
    rgb = (start + (end - start) * t).round().clip(0, 255).astype("uint8")
    return Image.fromarray(rgb, "RGB")


def chroma_key_logo(logo: Image.Image) -> Image.Image:
    """Make LOGO_BG_COLOR pixels transparent with a soft anti-aliased ramp."""
    arr = np.array(logo.convert("RGBA"))
    rgb = arr[..., :3].astype(np.int16)
    d = np.abs(rgb - np.array(LOGO_BG_COLOR)).max(axis=-1)
    ramp = np.clip(255 * (d - BG_THRESHOLD) / (BG_RAMP - BG_THRESHOLD), 0, 255)
    new_alpha = np.where(d < BG_THRESHOLD, 0, np.minimum(arr[..., 3], ramp))
    arr[..., 3] = new_alpha.astype("uint8")
    return Image.fromarray(arr, "RGBA")


def paste_logo(canvas: Image.Image) -> None:
    logo = chroma_key_logo(Image.open(LOGO_PATH))
    scale = LOGO_TARGET_HEIGHT / logo.height
    new_size = (round(logo.width * scale), LOGO_TARGET_HEIGHT)
    logo = logo.resize(new_size, Image.LANCZOS)
    y = (HEIGHT - logo.height) // 2
    canvas.paste(logo, (LOGO_LEFT_PADDING, y), logo)


def draw_text(canvas: Image.Image) -> None:
    draw = ImageDraw.Draw(canvas, "RGBA")
    wordmark_font = load_font(FONT_CANDIDATES_BOLD, WORDMARK_SIZE)
    tagline_font = load_font(FONT_CANDIDATES_REGULAR, TAGLINE_SIZE)
    bullets_font = load_font(FONT_CANDIDATES_REGULAR, BULLETS_SIZE)

    lines = [
        (WORDMARK, wordmark_font, (255, 255, 255, 255), WORDMARK_GAP),
        (TAGLINE, tagline_font, (255, 255, 255, 242), TAGLINE_GAP),
        (BULLETS, bullets_font, (255, 255, 255, 217), 0),
    ]
    measured = [(text, font, color, gap, font.getbbox(text)) for text, font, color, gap in lines]
    block_height = sum((bbox[3] - bbox[1] + gap) for _, _, _, gap, bbox in measured)
    y = (HEIGHT - block_height) // 2

    for text, font, color, gap, bbox in measured:
        draw.text((TEXT_LEFT, y), text, font=font, fill=color)
        y += (bbox[3] - bbox[1]) + gap


def main() -> int:
    canvas = render_gradient()
    paste_logo(canvas)
    draw_text(canvas)
    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(OUTPUT_PATH, format="PNG", optimize=True)
    print(f"wrote {OUTPUT_PATH.relative_to(REPO_ROOT)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
