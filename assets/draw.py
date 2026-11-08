import math, os, sys
from PIL import Image, ImageDraw, ImageFont, ImageFilter

OUT = sys.argv[1]
SS = 4
NAVY_D, NAVY = (11, 16, 51), (30, 46, 120)
GOLD, GOLD_D, CREAM = (246, 190, 52), (186, 124, 18), (255, 244, 214)
SECT = [(217, 74, 74), (43, 179, 163), (122, 79, 208), (47, 111, 237),
        (242, 138, 48), (91, 180, 80), (232, 90, 160), (246, 190, 52)]
FONT = "/System/Library/Fonts/Helvetica.ttc"


def canvas(n):
    im = Image.new("RGBA", (n * SS, n * SS), (0, 0, 0, 0))
    return im, ImageDraw.Draw(im)


def done(im, n):
    return im.resize((n, n), Image.LANCZOS)


def wheel(im, d, cx, cy, R, ang=0.0, pointer=True):
    d.ellipse([cx - R * 1.08, cy - R * 1.08 + R * .06, cx + R * 1.08, cy + R * 1.08 + R * .06], fill=(0, 0, 0, 70))
    d.ellipse([cx - R * 1.06, cy - R * 1.06, cx + R * 1.06, cy + R * 1.06], fill=GOLD_D)
    d.ellipse([cx - R * 1.02, cy - R * 1.02, cx + R * 1.02, cy + R * 1.02], fill=GOLD)
    r = R * .86
    for k, col in enumerate(SECT):
        a0 = ang - 112.5 + 45 * k
        d.pieslice([cx - r, cy - r, cx + r, cy + r], a0, a0 + 45, fill=col)
        x = cx + r * math.cos(math.radians(a0))
        y = cy + r * math.sin(math.radians(a0))
        d.line([cx, cy, x, y], fill=GOLD, width=int(R * .025))
    d.ellipse([cx - r, cy - r, cx + r, cy + r], outline=GOLD_D, width=int(R * .03))
    for k in range(16):
        a = math.radians(ang + 22.5 * k)
        x, y = cx + R * .94 * math.cos(a), cy + R * .94 * math.sin(a)
        lr = R * .035
        d.ellipse([x - lr, y - lr, x + lr, y + lr], fill=CREAM if k % 2 else GOLD_D)
    gloss = Image.new("RGBA", im.size, (0, 0, 0, 0))
    g = ImageDraw.Draw(gloss)
    g.ellipse([cx - r * .85, cy - r * .95, cx + r * .2, cy - r * .05], fill=(255, 255, 255, 46))
    mask = Image.new("L", im.size, 0)
    ImageDraw.Draw(mask).ellipse([cx - r, cy - r, cx + r, cy + r], fill=255)
    im.alpha_composite(Image.composite(gloss, Image.new("RGBA", im.size, (0, 0, 0, 0)), mask))
    for rr, col in ((R * .2, GOLD_D), (R * .17, GOLD), (R * .11, NAVY_D), (R * .07, NAVY)):
        d.ellipse([cx - rr, cy - rr, cx + rr, cy + rr], fill=col)
    if pointer:
        px, py, w, h = cx, cy - R * 1.12, R * .16, R * .3
        d.polygon([(px - w, py - h * .2), (px + w, py - h * .2), (px, py + h)], fill=NAVY_D)
        s = w * .72
        d.polygon([(px - s, py - h * .05), (px + s, py - h * .05), (px, py + h * .78)], fill=(255, 90, 80))


def star(d, cx, cy, ro, fill):
    pts = []
    for i in range(10):
        r = ro if i % 2 == 0 else ro * .42
        a = math.radians(-90 + 36 * i)
        pts.append((cx + r * math.cos(a), cy + r * math.sin(a)))
    d.polygon(pts, fill=fill)


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
    im = bg_gradient(n * SS)
    d = ImageDraw.Draw(im)
    wheel(im, d, n * SS / 2, n * SS * .53, n * SS * .33)
    done(im, n).convert("RGB").save(f"{OUT}/icon_640.png")
    done(im, 1024).convert("RGB").save(f"{OUT}/icon_1024.png")


def sticker_wheel(ang=0.0, n=512, pointer=True):
    im, d = canvas(n)
    wheel(im, d, n * SS / 2, n * SS * .54, n * SS * .4, ang, pointer)
    return done(im, n)


def sticker_pickle(n=512):
    im, d = canvas(n)
    S = n * SS
    pk = Image.new("RGBA", im.size, (0, 0, 0, 0))
    p = ImageDraw.Draw(pk)
    p.rounded_rectangle([S * .32, S * .08, S * .68, S * .92], radius=S * .17, fill=(40, 120, 45), outline=(20, 70, 25), width=int(S * .02))
    p.rounded_rectangle([S * .37, S * .12, S * .5, S * .86], radius=S * .06, fill=(95, 176, 80))
    for i, (x, y) in enumerate([(.45, .2), (.58, .28), (.43, .38), (.57, .47), (.46, .58), (.58, .66), (.5, .78), (.42, .72)]):
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
            ch = "П" if i == 0 else "Р"
            d.text((x + t / 2, y0 + t / 2), ch, font=f, fill=NAVY_D, anchor="mm")
        else:
            d.rounded_rectangle([x + t * .15, y0 + t * .12, x + t * .85, y0 + t * .35], radius=S * .01, fill=(255, 255, 255, 60))
    f2 = ImageFont.truetype(FONT, int(S * .075), index=1)
    d.text((S / 2, y0 + t + S * .1), "ДНЯ", font=f2, fill=GOLD, anchor="mm")
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


def spin(n=512, frames=36):
    fs = []
    for i in range(frames):
        a = 360 * i / frames
        fs.append(sticker_wheel(a, n))
    return fs


if __name__ == "__main__":
    os.makedirs(f"{OUT}/stickers", exist_ok=True)
    os.makedirs(f"{OUT}/frames", exist_ok=True)
    icon()
    for name, fn in (("drum", sticker_wheel), ("pickle", sticker_pickle), ("gift", sticker_gift),
                     ("board", sticker_board), ("trophy", sticker_trophy)):
        fn().save(f"{OUT}/stickers/{name}.png")
    fr = spin()
    for i, f in enumerate(fr):
        f.save(f"{OUT}/frames/{i:03d}.png")
    small = []
    for f in fr:
        bg = Image.new("RGBA", f.size, NAVY_D + (255,))
        bg.alpha_composite(f)
        small.append(bg.convert("P", palette=Image.ADAPTIVE).resize((256, 256)))
    small[0].save(f"{OUT}/drum_spin.gif", save_all=True, append_images=small[1:], duration=40, loop=0, disposal=2)
