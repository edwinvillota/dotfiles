# chart

Renders a CSV result set as a chart in the terminal, using
[plotext](https://github.com/piccolomo/plotext).

Installed to `~/.config/chart/` by `dotfiles`; driven by the `chart` shell
function in `zsh/config/db-clients.zsh`, which shares its connection lookup
with `db` so the two can never drift.

    chart dev 'select day, revenue from sales'   # query a NVIM_DB_* connection
    chart 'select ...'                           # when only one is defined
    chart -f report.sql                          # query from a file
    chart -i results.csv                         # a CSV file
    psql ... --csv | chart                       # CSV on stdin

    -t line|bar|barh|barstack|barhstack|scatter|hist|box   # default: auto
    -x COL      x axis column (default: the first)
    -y A,B      series columns (default: every numeric column)
    -T title    -W width   -H height

## Conventions

The **first column is the x axis** and every remaining numeric column is a
series, which matches how you would naturally write the SELECT.

Auto-detection picks: `hist` when there is nothing but the x column, `line`
when x is a date, `bar` when x is a label, and `line`/`bar` by row count when x
is numeric.

## Long vs wide

Results with two categorical dimensions arrive in **long** format:

    loadType,code,count
    new,M-087,1
    other,M-087,3

Charted as-is, `code` is silently dropped - the x axis just repeats
new/other/user. Pivot to **wide** so each series is its own column:

    select code,
           sum(count) filter (where "loadType" = 'new')   as new,
           sum(count) filter (where "loadType" = 'other') as other,
           sum(count) filter (where "loadType" = 'user')  as "user"
    from t group by code order by code;

(`user` needs quoting - it is a reserved word.) That charts as a grouped bar
with a legend. Use `-t barstack` instead when the question is each code's
*composition* rather than a type-by-type comparison.

For a handful of rows, a composite label and `-t barh` reads just as well and
needs no legend at all: `select code || ' ' || "loadType" as label, count`.

## Notes

- plotext is pip-only, so it lives in its own venv at
  `~/.local/share/chart/venv`, created on first run. `chart --setup` rebuilds
  it. Nothing is installed into the system python.
- `psql` is invoked with `--no-psqlrc`: `~/.psqlrc` switches output format on
  `ON_ERROR_STOP` and turns on `\timing`, none of which should reach the CSV
  parser. `--csv` is passed explicitly instead.
- The renderer targets **plotext 6.x**, whose API is object-based
  (`fig.signal(...)` then `fig.draw(...)`). Almost every example online is
  plotext 5 (`plt.plot()`), which no longer exists.
- Multi-series bars draw each series as its **own** bar signal, positioned by
  hand (side by side, or floating on a running baseline when stacked). plotext's
  own grouped/stacked bar takes every series in one call, and `signal.label()`
  accepts a single string, so that route can only ever yield one legend entry -
  identity by colour alone, which is not good enough.
- The legend is painted over the canvas, so space is reserved for it by widening
  an axis: columns on the right for bars, headroom at the top elsewhere. With
  negative values the baseline is not the axis start, the reservation cannot be
  computed, and the legend falls back to wherever plotext puts it.
- Series whose magnitudes differ by more than 50x trigger a warning: on a
  single axis the smaller one flattens into the baseline. Two y-axes are never
  the fix - plot them separately with `-y`.
