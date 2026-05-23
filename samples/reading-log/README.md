# reading-log

**A kid's reading list → a cheerful HTML certificate with a progress
bar toward a yearly goal.**

For parents and teachers. Encourages kids; no spreadsheets.

## What you write

```
log "Emma's reading log" age 7
    goal 500

    book "Charlotte's Web"           pages 184  rating 5
    book "The Wild Robot"            pages 277  rating 5
    book "Mr. Popper's Penguins"     pages 138  rating 4
    book "Frog and Toad Together"    pages 64   rating 5
    book "Junie B. Jones #1"         pages 72   rating 3
end
```

## What you get

A bright orange-and-cream "certificate" with:

- The kid's name and age
- A progress bar showing pages-read vs. yearly goal
- A table of every book with star ratings (★★★★☆)

## Update through the year

Re-run `capy run lib.capy script.capy` after adding new `book`
lines. Print quarterly to put on the fridge.
