#!/usr/bin/env python3
"""
Render a CSV result set as a terminal chart.

Reads CSV on stdin (or -i FILE) and picks a sensible chart, or takes an
explicit -t. Written against plotext 6.x, whose API is object-based
(fig.signal(...) -> fig.draw(...)) and NOT the fig.plot() style shown in most
online examples, which is plotext 5.

Column convention: the FIRST column is the x axis (labels, dates or numbers);
every remaining numeric column is a series.
"""
import argparse, csv, datetime, shutil, sys

def die(msg, code=2):
    print(f"chart: {msg}", file=sys.stderr)
    sys.exit(code)

DATE_FORMATS = ("%Y-%m-%d", "%Y-%m-%d %H:%M:%S", "%Y-%m-%dT%H:%M:%S",
                "%Y-%m", "%d/%m/%Y", "%Y/%m/%d")

def as_float(v):
    if v is None:
        return None
    s = v.strip().replace(",", "")
    if s == "" or s.lower() in ("null", "none", "∅", "nan"):
        return None
    try:
        return float(s)
    except ValueError:
        return None

def as_date(v):
    if not v:
        return None
    s = v.strip()
    for f in DATE_FORMATS:
        try:
            return datetime.datetime.strptime(s, f)
        except ValueError:
            pass
    return None

def column_kind(values):
    """numeric | date | label — based on what the whole column parses as."""
    vals = [v for v in values if v is not None and v.strip() != ""]
    if not vals:
        return "label"
    if all(as_float(v) is not None for v in vals):
        return "numeric"
    if all(as_date(v) is not None for v in vals):
        return "date"
    return "label"

def sniff_delimiter(sample):
    """psql --csv gives commas, but a pasted file may not. Score candidates by
    field-count consistency, the same approach as the visidata rc and
    csvview.nvim config."""
    best, best_score = ",", 0.0
    for d in (",", ";", "\t", "|"):
        try:
            rows = [r for r in csv.reader(sample.splitlines(), delimiter=d) if r]
        except Exception:
            continue
        if not rows:
            continue
        counts = [len(r) for r in rows]
        n = max(set(counts), key=counts.count)
        if n < 2:
            continue
        score = counts.count(n) / len(counts) * n
        if score > best_score:
            best, best_score = d, score
    return best

def read_csv(fp):
    text = fp.read()
    if not text.strip():
        die("no input (empty result set?)")
    delim = sniff_delimiter(text[:8192])
    rows = list(csv.reader(text.splitlines(), delimiter=delim))
    rows = [r for r in rows if any(c.strip() for c in r)]
    if not rows:
        die("no rows")
    header, body = rows[0], rows[1:]
    if not body:
        die("result set has a header but no rows")
    width = len(header)
    body = [r + [""] * (width - len(r)) for r in body]
    cols = {h: [r[i] if i < len(r) else "" for r in body] for i, h in enumerate(header)}
    return header, cols, len(body)

def pick_type(kinds, nseries, nrows):
    xk = kinds[0]
    if nseries == 0:
        return "hist"          # nothing but the x column -> its distribution
    if xk == "date":
        return "line"
    if xk == "label":
        return "bar"
    return "line" if nrows > 40 else "bar"

# dataviz reference palette, dark-surface categorical steps
PALETTE = [(57,135,229), (217,89,38), (25,158,112), (201,133,0),
           (213,81,129), (0,131,0), (144,133,233), (230,103,103)]

def main():
    ap = argparse.ArgumentParser(prog="chart", add_help=True)
    ap.add_argument("-i", "--input", help="CSV file (default: stdin)")
    ap.add_argument("-t", "--type",
                    choices=["auto","line","bar","barh","scatter","hist","box"],
                    default="auto")
    ap.add_argument("-T", "--title", default=None)
    ap.add_argument("-W", "--width", type=int, default=None)
    ap.add_argument("-H", "--height", type=int, default=None)
    ap.add_argument("-x", "--xcol", help="column to use for x (default: first)")
    ap.add_argument("-y", "--ycols", help="comma-separated series columns")
    ap.add_argument("--theme", default="dark")
    args = ap.parse_args()

    fp = open(args.input, encoding="utf-8") if args.input else sys.stdin
    header, cols, nrows = read_csv(fp)

    xname = args.xcol or header[0]
    if xname not in cols:
        die(f"no column {xname!r}; have: {', '.join(header)}")
    if args.ycols:
        ynames = [c.strip() for c in args.ycols.split(",")]
        missing = [c for c in ynames if c not in cols]
        if missing:
            die(f"no column(s) {', '.join(missing)}; have: {', '.join(header)}")
    else:
        ynames = [h for h in header
                  if h != xname and column_kind(cols[h]) == "numeric"]

    # Series on wildly different scales share one axis and the small one
    # flattens into the baseline - a chart that looks fine and reads wrong.
    # Warn rather than silently draw it; two axes is never the answer.
    if len(ynames) > 1:
        spans = {}
        for y in ynames:
            vals = [v for v in (as_float(s) for s in cols[y]) if v is not None]
            if vals:
                spans[y] = max(abs(v) for v in vals) or 1.0
        if spans and max(spans.values()) / min(spans.values()) > 50:
            big = max(spans, key=spans.get)
            small = min(spans, key=spans.get)
            print(f"chart: {big!r} and {small!r} differ by "
                  f"{max(spans.values())/min(spans.values()):.0f}x - {small!r} will "
                  f"flatten against the axis.\n"
                  f"       plot them separately with -y, e.g. -y {big} / -y {small}",
                  file=sys.stderr)

    kinds = [column_kind(cols[xname])] + [column_kind(cols[y]) for y in ynames]
    ctype = args.type if args.type != "auto" else pick_type(kinds, len(ynames), nrows)

    try:
        import plotext as plt
    except ImportError:
        die("plotext is not installed in the chart venv (try: chart --setup)")

    fig = plt.figure
    fig.clear()
    fig.theme(args.theme)
    # shutil.get_terminal_size takes the fallback; os.get_terminal_size takes a
    # file descriptor. It also honours $COLUMNS/$LINES and works when stdout is
    # a pipe, so there is no isatty() branch to get wrong.
    term = shutil.get_terminal_size((100, 30))
    fig.plot_size(args.width or max(40, min(term.columns - 2, 200)),
                  args.height or max(12, min(term.lines - 4, 50)))

    title = args.title or (f"{', '.join(ynames)} by {xname}" if ynames else xname)

    def mark(i, symbol="braille"):
        return plt.marker(symbol=symbol, pixel=PALETTE[i % len(PALETTE)])

    xk = kinds[0]
    if ctype == "hist":
        target = ynames[0] if ynames else xname
        data = [v for v in (as_float(s) for s in cols[target]) if v is not None]
        if not data:
            die(f"column {target!r} has no numeric values to bin")
        fig.draw(fig.hist(data, bins=min(40, max(8, len(data)//4))))
        title = args.title or f"distribution of {target}"

    elif ctype == "box":
        series = [[v for v in (as_float(s) for s in cols[y]) if v is not None]
                  for y in ynames]
        if not series:
            die("box needs at least one numeric column")
        fig.draw(fig.box(ynames, series))

    elif ctype in ("bar", "barh"):
        labels = cols[xname]
        if not ynames:
            die("bar needs a numeric column besides the x column")
        heights = [[as_float(v) or 0.0 for v in cols[y]] for y in ynames]
        marks = [plt.marker(symbol="full", pixel=PALETTE[i % len(PALETTE)])
                 for i in range(len(ynames))]
        kw = {"marker": marks if len(ynames) > 1 else marks[0]}
        if ctype == "barh":
            kw["orientation"] = "h"
        bars = fig.bar(labels, heights if len(ynames) > 1 else heights[0], **kw)
        fig.draw(bars)

    else:  # line / scatter
        if not ynames:
            die("need a numeric column to plot; got only " + xname)
        if xk == "date":
            fig.date("x").activate()
            xs = [as_date(v) for v in cols[xname]]
        elif xk == "numeric":
            xs = [as_float(v) for v in cols[xname]]
        else:
            xs = list(range(1, nrows + 1))
        for i, y in enumerate(ynames):
            ys = [as_float(v) for v in cols[y]]
            pairs = [(a, b) for a, b in zip(xs, ys) if a is not None and b is not None]
            if not pairs:
                continue
            px, py = [p[0] for p in pairs], [p[1] for p in pairs]
            s = fig.signal(px, py,
                           marker=mark(i, "dot" if ctype == "scatter" else "braille"))
            s.label(y)
            if ctype == "line":
                s.lines()
            fig.draw(s)
        fig.label(xname, axis="x")

    fig.title(title)
    print(fig.build())

if __name__ == "__main__":
    main()
