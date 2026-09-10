#!/usr/bin/env python3
"""Regenerates windows-us-international-keyboard-without-dead-keys.ico. Requires Pillow (pip install Pillow) and a
DejaVu Sans Bold font (fonts-dejavu-core on Debian/Ubuntu). After running
this, recompile the resource with:

    x86_64-w64-mingw32-windres -O coff -o windows-us-international-keyboard-without-dead-keys_windows_amd64.syso rsrc.rc
"""
from PIL import Image, ImageDraw, ImageFont

FONT_PATH = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"


def draw_key(size, letter="A"):
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    pad = size * 0.08
    r = size * 0.16
    shadow_off = size * 0.045
    d.rounded_rectangle(
        [pad + shadow_off, pad + shadow_off, size - pad + shadow_off, size - pad + shadow_off],
        radius=r, fill=(0, 0, 0, 70),
    )
    # Keycap body (dark)
    d.rounded_rectangle([pad, pad, size - pad, size - pad], radius=r, fill=(51, 58, 74, 255))
    # Keycap top face (inset, lighter) - the "well" of the key
    inset = size * 0.09
    top_bottom = size - pad - inset * 1.6
    d.rounded_rectangle(
        [pad + inset, pad + inset * 0.7, size - pad - inset, top_bottom],
        radius=r * 0.7, fill=(240, 242, 246, 255),
    )
    # Letter glyph
    font_size = int((top_bottom - (pad + inset * 0.7)) * 0.78)
    font = ImageFont.truetype(FONT_PATH, font_size)
    bbox = d.textbbox((0, 0), letter, font=font)
    tw, th = bbox[2] - bbox[0], bbox[3] - bbox[1]
    cx = size / 2
    cy = (pad + inset * 0.7 + top_bottom) / 2
    d.text((cx - tw / 2 - bbox[0], cy - th / 2 - bbox[1]), letter, font=font, fill=(51, 58, 74, 255))
    return img


if __name__ == "__main__":
    sizes = [16, 24, 32, 48, 64, 128, 256]
    base = draw_key(256)
    imgs = [base.resize((s, s), Image.LANCZOS) if s != 256 else base for s in sizes]
    imgs[0].save("windows-us-international-keyboard-without-dead-keys.ico", format="ICO", sizes=[(s, s) for s in sizes], append_images=imgs[1:])
    print("wrote windows-us-international-keyboard-without-dead-keys.ico")
