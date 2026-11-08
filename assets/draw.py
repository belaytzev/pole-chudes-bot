"""Draws the bot's neutral visual set: profile icon, object stickers and the spinning-drum frames.

    uv run --with pillow python assets/draw.py <out-dir>

Writes <out-dir>/icon.png, <out-dir>/stickers/*.png and <out-dir>/frames/*.png (encode the frames with ffmpeg).
"""
import math, os, sys
from PIL import Image, ImageDraw, ImageFont

OUT = sys.argv[1]
SS = 4
NAVY_D, NAVY = (11, 16, 51), (30, 46, 120)
GOLD, GOLD_D, CREAM = (246, 190, 52), (186, 124, 18), (255, 244, 214)
BLACK, WHITE = (14, 14, 26), (246, 246, 252)
SPECIAL = {0: (217, 74, 74), 5: (246, 190, 52), 11: (47, 111, 237), 17: (91, 180, 80), 23: (217, 74, 74),
           29: (246, 190, 52), 34: (122, 79, 208)}
FONT = "/System/Library/Fonts/Helvetica.ttc"
FRAMES, TILT, THICK = 48, 0.58, 22


def canvas(n):
    im = Image.new("RGBA", (n * SS, n * SS), (0, 0, 0, 0))
    return im, ImageDraw.Draw(im)


def done(im, n):
    return im.resize((n, n), Image.LANCZOS)


def star(d, cx, cy, ro, fill):
    pts = []
    for i in range(10):
        r = ro if i % 2 == 0 else ro * .42
        a = math.radians(-90 + 36 * i)
        pts.append((cx + r * math.cos(a), cy + r * math.sin(a)))
    d.polygon(pts, fill=fill)


def disc(size, ang):
    """The drum seen from above: 36 black-and-white sectors, a gold rim with bulbs, a pointer arrow on the hub."""
    im = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(im)
    c, R = size / 2, size * 0.47
    d.ellipse([c - R * 1.03, c - R * 1.03, c + R * 1.03, c + R * 1.03], fill=GOLD_D)
    d.ellipse([c - R, c - R, c + R, c + R], fill=GOLD)
    r = R * 0.86
    for k in range(36):
        a0 = ang + 10 * k - 90
        d.pieslice([c - r, c - r, c + r, c + r], a0, a0 + 10, fill=SPECIAL.get(k, BLACK if k % 2 == 0 else WHITE))
    d.ellipse([c - r, c - r, c + r, c + r], outline=GOLD_D, width=max(2, int(R * 0.02)))
    for k in range(24):
        a = math.radians(15 * k)
        x, y = c + R * 0.93 * math.cos(a), c + R * 0.93 * math.sin(a)
        lr = R * 0.032
        d.ellipse([x - lr, y - lr, x + lr, y + lr], fill=CREAM)
    gloss = Image.new("RGBA", im.size, (0, 0, 0, 0))
    ImageDraw.Draw(gloss).ellipse([c - r * 0.9, c - r * 0.98, c + r * 0.1, c - r * 0.1], fill=(255, 255, 255, 38))
    mask = Image.new("L", im.size, 0)
    ImageDraw.Draw(mask).ellipse([c - r, c - r, c + r, c + r], fill=255)
    im.alpha_composite(Image.composite(gloss, Image.new("RGBA", im.size, (0, 0, 0, 0)), mask))
    for rr, col in ((R * 0.17, GOLD_D), (R * 0.145, GOLD), (R * 0.09, (255, 226, 130))):
        d.ellipse([c - rr, c - rr, c + rr, c + rr], fill=col)
    a = math.radians(-38)
    ux, uy = math.cos(a), math.sin(a)
    px, py = -uy, ux
    tail, neck, tip, sw, hw = R * 0.06, R * 0.5, R * 0.68, R * 0.035, R * 0.1
    pts = [(c + ux * tail + px * sw, c + uy * tail + py * sw), (c + ux * neck + px * sw, c + uy * neck + py * sw),
           (c + ux * neck + px * hw, c + uy * neck + py * hw), (c + ux * tip, c + uy * tip),
           (c + ux * neck - px * hw, c + uy * neck - py * hw), (c + ux * neck - px * sw, c + uy * neck - py * sw),
           (c + ux * tail - px * sw, c + uy * tail - py * sw)]
    d.polygon(pts, fill=GOLD, outline=GOLD_D)
    return im


def drum(ang, w=512):
    """A tilted drum with a gold side wall, w pixels wide, transparent background."""
    n = w * SS
    top = disc(n, ang).resize((n, round(n * TILT)), Image.LANCZOS)
    th = THICK * SS
    out = Image.new("RGBA", (n, top.height + th + 8 * SS), (0, 0, 0, 0))
    base = top.getchannel("A")
    for i in range(th, 0, -1):
        layer = Image.new("RGBA", top.size, GOLD_D + (255,))
        layer.putalpha(base)
        out.alpha_composite(layer, (0, 4 * SS + i))
    out.alpha_composite(top, (0, 4 * SS))
    return out.resize((w, round(out.height / SS)), Image.LANCZOS)


def bg_gradient(n):
    im = Image.new("RGBA", (n, n), NAVY_D + (255,))
    d = ImageDraw.Draw(im)
    for i in range(60, 0, -1):
        t = i / 60
        c = tuple(int(NAVY_D[j] + (NAVY[j] - NAVY_D[j]) * (1 - t) ** 1.3) for j in range(3))
        r = n * .75 * t
        d.ellipse([n / 2 - r, n * .45 - r, n / 2 + r, n * .45 + r], fill=c + (255,))
    for sx, sy, sr in [(.12, .14, 9), (.88, .2, 7), (.18, .84, 6), (.86, .8, 10), (.5, .06, 5), (.07, .5, 5), (.94, .52, 6)]:
        star(d, sx * n, sy * n, sr * n / 640 * 1.4, GOLD + (255,))
    return im


def icon():
    n = 640
    im = bg_gradient(n)
    dr = drum(20, 540)
    im.alpha_composite(dr, ((n - dr.width) // 2, (n - dr.height) // 2))
    im.convert("RGB").save(f"{OUT}/icon.png", optimize=True)


def sticker_pickle(n=512):
    im, d = canvas(n)
    S = n * SS
    pk = Image.new("RGBA", im.size, (0, 0, 0, 0))
    p = ImageDraw.Draw(pk)
    p.rounded_rectangle([S * .32, S * .08, S * .68, S * .92], radius=S * .17, fill=(40, 120, 45), outline=(20, 70, 25), width=int(S * .02))
    p.rounded_rectangle([S * .37, S * .12, S * .5, S * .86], radius=S * .06, fill=(95, 176, 80))
    for x, y in [(.45, .2), (.58, .28), (.43, .38), (.57, .47), (.46, .58), (.58, .66), (.5, .78), (.42, .72)]:
        p.ellipse([S * x - S * .014, S * y - S * .014, S * x + S * .014, S * y + S * .014], fill=(20, 80, 28))
    p.ellipse([S * .4, S * .3, S * .46, S * .36], fill=(255, 255, 255))
    p.ellipse([S * .54, S * .3, S * .6, S * .36], fill=(255, 255, 255))
    p.ellipse([S * .42, S * .32, S * .45, S * .35], fill=NAVY_D)
    p.ellipse([S * .56, S * .32, S * .59, S * .35], fill=NAVY_D)
    p.arc([S * .42, S * .36, S * .58, S * .5], 20, 160, fill=NAVY_D, width=int(S * .014))
    pk = pk.rotate(28, resample=Image.BICUBIC, center=(S / 2, S / 2))
    im.alpha_composite(pk)
    return done(im, n)


def sticker_gift(n=512):
    im, d = canvas(n)
    S = n * SS
    d.rounded_rectangle([S * .14, S * .42, S * .86, S * .88], radius=S * .03, fill=(217, 74, 74), outline=NAVY_D, width=int(S * .015))
    d.rounded_rectangle([S * .1, S * .3, S * .9, S * .46], radius=S * .03, fill=(236, 100, 100), outline=NAVY_D, width=int(S * .015))
    d.rectangle([S * .45, S * .3, S * .55, S * .88], fill=GOLD)
    d.ellipse([S * .26, S * .12, S * .5, S * .34], outline=GOLD, width=int(S * .045))
    d.ellipse([S * .5, S * .12, S * .74, S * .34], outline=GOLD, width=int(S * .045))
    d.ellipse([S * .44, S * .24, S * .56, S * .36], fill=GOLD_D)
    for x, y in ((.18, .14), (.84, .2), (.7, .07)):
        star(d, S * x, S * y, S * .04, GOLD + (255,))
    return done(im, n)


def sticker_board(n=512):
    im, d = canvas(n)
    S = n * SS
    f = ImageFont.truetype(FONT, int(S * .12), index=1)
    t, g, x0, y0 = S * .165, S * .025, S * .04, S * .38
    for i in range(5):
        x = x0 + i * (t + g)
        edge = i in (0, 4)
        d.rounded_rectangle([x, y0 + S * .02, x + t, y0 + t + S * .02], radius=S * .02, fill=(0, 0, 0, 80))
        d.rounded_rectangle([x, y0, x + t, y0 + t], radius=S * .02, fill=GOLD if edge else (47, 111, 237), outline=NAVY_D, width=int(S * .01))
        if edge:
            d.text((x + t / 2, y0 + t / 2), "П" if i == 0 else "Р", font=f, fill=NAVY_D, anchor="mm")
        else:
            d.rounded_rectangle([x + t * .15, y0 + t * .12, x + t * .85, y0 + t * .35], radius=S * .01, fill=(255, 255, 255, 60))
    d.text((S / 2, y0 + t + S * .1), "ДНЯ", font=ImageFont.truetype(FONT, int(S * .075), index=1), fill=GOLD, anchor="mm")
    return done(im, n)


def sticker_trophy(n=512):
    im, d = canvas(n)
    S = n * SS
    d.arc([S * .1, S * .14, S * .4, S * .48], 90, 270, fill=GOLD_D, width=int(S * .04))
    d.arc([S * .6, S * .14, S * .9, S * .48], 270, 90, fill=GOLD_D, width=int(S * .04))
    d.polygon([(S * .26, S * .1), (S * .74, S * .1), (S * .68, S * .46), (S * .56, S * .6), (S * .44, S * .6), (S * .32, S * .46)], fill=GOLD, outline=NAVY_D)
    d.polygon([(S * .3, S * .13), (S * .4, S * .13), (S * .38, S * .43), (S * .34, S * .4)], fill=(255, 230, 140))
    d.rectangle([S * .46, S * .58, S * .54, S * .74], fill=GOLD_D)
    d.rounded_rectangle([S * .32, S * .72, S * .68, S * .88], radius=S * .02, fill=NAVY, outline=GOLD, width=int(S * .015))
    star(d, S * .5, S * .31, S * .1, NAVY_D + (255,))
    return done(im, n)


if __name__ == "__main__":
    os.makedirs(f"{OUT}/stickers", exist_ok=True)
    os.makedirs(f"{OUT}/frames", exist_ok=True)
    icon()
    drum(20).save(f"{OUT}/stickers/drum.png", optimize=True)
    for name, fn in (("pickle", sticker_pickle), ("gift", sticker_gift), ("board", sticker_board), ("trophy", sticker_trophy)):
        fn().save(f"{OUT}/stickers/{name}.png", optimize=True)
    for i in range(FRAMES):
        drum(360 * i / FRAMES).save(f"{OUT}/frames/{i:03d}.png")
