"""Renders the spinning game-show drum: a flat black-and-white disc seen at an angle, a gold rim with bulbs
and a gold pointer arrow on the hub. Writes transparent PNG frames; encode them with ffmpeg."""
import math, os, sys
from PIL import Image, ImageDraw

OUT = sys.argv[1]
FRAMES = 48
SS = 3
W = 512
TILT = 0.58
THICK = 22
GOLD, GOLD_D, CREAM = (246, 190, 52), (176, 112, 14), (255, 244, 214)
BLACK, WHITE = (14, 14, 26), (246, 246, 252)
SPECIAL = {0: (217, 74, 74), 5: (246, 190, 52), 11: (47, 111, 237), 17: (91, 180, 80), 23: (217, 74, 74),
           29: (246, 190, 52), 34: (122, 79, 208)}


def disc(size, ang):
    im = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(im)
    c, R = size / 2, size * 0.47
    d.ellipse([c - R * 1.03, c - R * 1.03, c + R * 1.03, c + R * 1.03], fill=GOLD_D)
    d.ellipse([c - R * 1.0, c - R * 1.0, c + R * 1.0, c + R * 1.0], fill=GOLD)
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


def frame(ang):
    n = W * SS
    top = disc(n, ang).resize((n, round(n * TILT)), Image.LANCZOS)
    th = THICK * SS
    H = top.height + th + 8 * SS
    out = Image.new("RGBA", (n, H), (0, 0, 0, 0))
    base = top.getchannel("A")
    for i in range(th, 0, -1):
        layer = Image.new("RGBA", top.size, GOLD_D + (255,))
        layer.putalpha(base)
        out.alpha_composite(layer, (0, 4 * SS + i))
    out.alpha_composite(top, (0, 4 * SS))
    return out.resize((W, round(H / SS)), Image.LANCZOS)


if __name__ == "__main__":
    os.makedirs(OUT, exist_ok=True)
    for i in range(FRAMES):
        frame(360 * i / FRAMES).save(f"{OUT}/{i:03d}.png")
    print("frames", FRAMES, frame(0).size)
