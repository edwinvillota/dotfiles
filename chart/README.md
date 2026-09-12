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

    -t line|bar|barh|scatter|hist|box            # default: auto
    -x COL      x axis column (default: the first)
    -y A,B      series columns (default: every numeric column)
    -T title    -W width   -H height

## Conventions

The **first column is the x axis** and every remaining numeric column is a
series, which matches how you would naturally write the SELECT.

Auto-detection picks: `hist` when there is nothing but the x column, `line`
when x is a date, `bar` when x is a label, and `line`/`bar` by row count when x
is numeric.

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
- Series whose magnitudes differ by more than 50x trigger a warning: on a
  single axis the smaller one flattens into the baseline. Two y-axes are never
  the fix - plot them separately with `-y`.
