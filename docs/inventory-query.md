# Inventory query language

The inventory table is filtered and sorted by **one query string**. The TUI
query bar, keyboard shortcuts (`p`, `f`, `r`, `o`, `O`, `x`), and palette
commands (`aws`, `running`, `clear`, …) all read and write that string.

Nothing else may implement its own filter or sort. Parse the string with
`internal/query`, then apply it with `query.Apply`. A later chip UI or the
graphical client should render the same `query.Query` — not a second matcher.

## The bar

One line above the table, full window width, using
[`charm.land/bubbles/v2/textinput`](https://pkg.go.dev/charm.land/bubbles/v2/textinput):

```
/ search, or provider=aws · name+ to sort                      11/20 vms

 NAME            PROVIDER   REGION        TYPE         STATE
```

- `/` focuses the bar; type freely; results update as you type.
- `enter` or `esc` returns focus to the table. The query **stays**.
- While the bar is focused, letters go into the query — including `p` / `f` /
  `o` and tab keys `1` / `2`. Shortcuts only fire when the table is focused.
- `x` (while the table is focused) clears the query.
- The right-hand count is `matching / loaded`.

## Grammar

The query is a list of whitespace-separated tokens. Double quotes group a
token that contains spaces: `"web api"`.

```
query  := token*
token  := filter | sort | search
filter := field "=" value
sort   := field direction | direction field
search := any other token
```

Spaces around `=` are allowed (`state = running` is the same as
`state=running`).

### Search (free text)

A token with no `=` and no sort marker is a **substring** search
(case-insensitive) across name, id, provider, region, type, and state.

Several search tokens are **AND**: every term must match somewhere.

```
web                 → name/id/… contains "web"
web prod            → contains "web" AND contains "prod"
"web api"           → contains "web api"
```

Search is not exact. It will match `us-east-1` if you type `us-east`.

### Filters (`field=value`)

`=` means **exact match** of that field (case-insensitive). `region=us` does
**not** match `us-east-1`. Use search for partial matches; use `=` when you
know the full value.

Several filters on **different** fields are **AND**.
Several filters on the **same** field are **OR**.

```
provider=aws
p=aws
provider=aws state=running
provider=aws provider=gcp     → aws OR gcp
p=aws web                     → provider is aws AND search "web"
```

| Field      | Aliases | Matches                         |
| ---------- | ------- | ------------------------------- |
| `name`     | `n`     | VM name                         |
| `provider` | `p`     | `aws`, `azure`, `gcp`, …        |
| `region`   | `r`     | provider region id              |
| `type`     | `t`     | machine type / size             |
| `state`    | `s`     | `running`, `stopped`, …         |
| `id`       |         | provider instance id            |
| `tag`      |         | `tag=key:value` (exact on both) |

Unknown `foo=bar` is **not** a filter; it is treated as search text so a typo
does not silently hide every row.

### Sort (column + direction)

Exactly one sort, taken from the **last** sort token. Direction is glued to
the column name with `+` (asc) or `-` (desc):

| Meaning     | Preferred | Also accepted              |
| ----------- | --------- | -------------------------- |
| ascending   | `name+`   | `+name`, `name:asc`        |
| descending  | `name-`   | `-name`, `name:desc`       |

Any filter field except `tag` can be a sort column (`name`, `provider`,
`region`, `type`, `state`, `id`, plus aliases).

```
name+
provider=aws name-
web p=gcp region+
```

No sort token means keep load order (stable).

## Combining — one example

```
api p=aws state=running region=us-east-1 name+
```

1. Search `"api"` in the haystack.
2. Provider **is** `aws`.
3. State **is** `running`.
4. Region **is** `us-east-1` (not `us-east-2`).
5. Sort by name ascending.

## Keyboard sugar

These keys only rewrite the query string, then run the same parse/apply path.

| Key | Effect |
| --- | --- |
| `/` | focus the query bar |
| `p` | cycle `provider=` through configured clouds, then remove it |
| `f` | cycle `state=` through running → stopped → starting → stopping → unknown → off |
| `r` | cycle `region=` through regions present in the loaded inventory, then off |
| `o` | cycle sort column: name → provider → region → type → state → off |
| `O` | flip sort direction (no-op when there is no sort) |
| `x` | clear the whole query |

Palette commands `aws` / `azure` / `gcp` / `all` / `running` / `stopped` /
`clear` / `region` do the same kind of rewrite.

## Implementation

| Piece | Package |
| ----- | ------- |
| Parse, format, cycle helpers | `internal/query` |
| Match + sort `[]core.VM` | `internal/query.Apply` |
| Textinput + key rewriting | `internal/tui` inventory |

Do not add `fProvider` / `sortKey` fields, and do not filter inside the table
widget. If a new UI needs chips, parse the string (or keep a `query.Query`)
and render chips from it.

### Live vs commit

Typing applies immediately. `esc` / `enter` only move focus. There is no
separate “pending” buffer — that was the old `/` search line, and it diverged
from `p`/`f` filters.
