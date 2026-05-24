# Library reference (→ `.html`)

Recipe DSL for home cooks. Six keywords (recipe, serves, time, ingredient, step, tip) produce a polished printable HTML recipe card. Anyone can use it within five minutes — no programming background required.

| | |
|---|---|
| **Output extension** | `.html` |
| **Functions** | 7 |
| **Types** | 0 |

## Functions

### `end`

```
end
```

### `ingredient`

Add one ingredient to the recipe. Listed in a two-column grid above the method.

```
ingredient <name> <qty>
```

| Argument | Type | Description |
|---|---|---|
| `name` | `string` | Ingredient name, e.g. `"olive oil"`. |
| `qty` | `string` | Quantity with unit, e.g. `"3/4 cup"`. |

### `recipe`

Open a new recipe with a title. Wraps the rest of the file; closed by `end`.

```
recipe <title>
```

| Argument | Type | Description |
|---|---|---|
| `title` | `string` | Display name of the dish, shown as the H1. |

**Opens an indented block** — body runs until `end`.

### `serves`

How many portions the recipe makes.

```
serves <n>
```

| Argument | Type | Description |
|---|---|---|
| `n` | `any` | Number of servings (bare number, e.g. `serves 8`). |

### `step`

Add one numbered step to the method. Render in source order.

```
step <text>
```

| Argument | Type | Description |
|---|---|---|
| `text` | `string` | What the cook should do at this step. |

### `time`

Total cooking time, shown in the meta line under the title.

```
time <t>
```

| Argument | Type | Description |
|---|---|---|
| `t` | `string` | Free-form time string, e.g. `"45 minutes"`. |

### `tip`

Optional tip box highlighted at the bottom of the card. Multiple tips render as separate boxes.

```
tip <text>
```

| Argument | Type | Description |
|---|---|---|
| `text` | `string` | Suggestion, variation, or chef's note. |

