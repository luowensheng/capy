# transpile-express-server

Route DSL → complete Node/Express server file. Pass to `node`
directly.

```sh
../../capy run lib.yaml script.capy > server.js
node server.js
```

5 lines of source → a ~25-line Express server with middleware, JSON
parsing, four routes, and a listener.
