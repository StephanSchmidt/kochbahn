<p align="center">
  <img src="assets/logo-blue.png" alt="kochbahn" width="420">
</p>

<p align="center">
  <a href="https://github.com/StephanSchmidt/kochbahn/actions/workflows/ci.yml"><img src="https://github.com/StephanSchmidt/kochbahn/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
</p>

Renders a multi-lane recipe from a YAML config to a top-down SVG timeline
suitable for printing in a book. Each **lane** is a parallel cooking process;
lanes can **fork** (one process splits) and **merge** (processes converge, e.g.
*"Nudeln in Pfanne"*). The vertical axis is time in minutes.

The name is German — *Koch* ("cook") + *Bahn* ("lane/track") — the project's
central abstraction. Recipe content and labels in the examples are German.

<p align="center">
  <img src="examples/lasagne.svg" alt="kochbahn timeline: a lasagne recipe with a fork and three merges across five lanes" width="640">
</p>

## Install

```
go install github.com/StephanSchmidt/kochbahn@latest
```

Or build from source: `make build` (produces a `kochbahn` binary).

## Usage

```
go run . -in examples/salbeibutter.yaml -out salbei.svg
go run . -in recipe.yaml            # -> recipe.svg
go run . -in recipe.yaml -out -     # SVG to stdout
```

Installed as a binary, replace `go run .` with `kochbahn`.

## Examples

Persistent, editable examples live in `examples/` (YAML + rendered SVG), spanning
simple to complex:

| File | Lanes | Shows |
|------|-------|-------|
| `examples/salbeibutter.yaml` | 3 | prep lane + two merges (Nudelwasser, then Nudeln) |
| `examples/fruehstueck.yaml`  | 3 | three lanes converging on a plate |
| `examples/lasagne.yaml`      | 5 | a fork + three merges, all colors |

Re-render them all after an edit:

```
make examples      # renders examples/*.yaml -> examples/*.svg
```

Time is shown per step as a **relative pill** above each node: the earliest action
reads `Anfang`; a lane's first action reads `+N min` from the start (when to fire
it up); later actions read `+N min` since that lane's previous step.

## Config format

```yaml
title: "Salbeibutter-Pasta"
subtitle: "2 Portionen"
time: { from: 0, to: 9, tick: 1, unit: "min" }   # optional; derived from steps

lanes:                       # order = left→right columns; max 5
  - id: nudeln
    label: "Wasser / Nudeln"
    color: "#3b82f6"         # optional; palette fills the rest
  - id: sosse
    label: "Soße"

steps:
  - lane: nudeln
    at: 0
    text: "Nudeln rein"      # \n in text = line break
  - lane: nudeln
    at: 7
    text: "abgießen"
    merge_into: sosse        # this lane converges into 'sosse'
    arrow_label: "Nudeln in Pfanne"
  # fork_to: [laneA, laneB]  # one step splitting into several lanes
```

## Architecture

A strict three-stage pipeline, each stage ignorant of the next's concerns:

```
YAML ──▶ recipe model ──▶ layout model ──▶ SVG bytes
         (semantics)      (geometry)        (render)
```

- `internal/recipe` — domain model + YAML load/validate. No pixels.
- `internal/layout` — renderer-agnostic geometry (`Build(recipe, Style) → Layout`
  of rails/nodes/labels/connectors/ticks/headers). No SVG.
- `internal/render` — maps a `Layout` to SVG. No recipe semantics.

A future PNG/canvas backend only needs to consume the `layout.Layout`.

## Test

```
go test ./...
```

Fixtures: `testdata/salbeibutter.yaml` (2 lanes, one merge),
`testdata/fork-merge.yaml` (5 lanes, a fork and three merges — exercises every
feature).

## License

[MIT](LICENSE) © 2026 Stephan Schmidt
