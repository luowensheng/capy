# recipe-card

**A recipe written in plain English → a polished HTML recipe card.**

For home cooks, food bloggers, and anyone who's tired of pasting
into a clunky recipe-website editor. No code knowledge required.

## What you write

```
recipe "Lemon olive oil cake"
    serves 8
    time "45 minutes"

    ingredient "all-purpose flour"   "1 1/2 cups"
    ingredient "olive oil"           "3/4 cup"
    ingredient "sugar"               "1 cup"
    ...

    step "Preheat oven to 350F and grease a 9-inch round pan."
    step "Whisk flour, baking powder, and salt in a bowl."
    ...

    tip "For extra zing, glaze with powdered sugar and lemon juice."
end
```

The words `recipe`, `serves`, `time`, `ingredient`, `step`, `tip`,
and `end` are the entire vocabulary. Anyone can learn it in a
minute.

## What you get

A standalone HTML file — gorgeous typography, two-column ingredient
list, numbered steps, highlighted tip box. Open it in a browser,
print it, email it, paste it into a recipe site.

## Run

```sh
capy run lib.capy script.capy > my-recipe.html
open my-recipe.html
```

## Make it yours

To restyle every recipe (different colors, fonts, layout), edit the
`<style>` block in `lib.capy` once. Every recipe regenerates with
the new look — no need to touch any of the recipes themselves.
