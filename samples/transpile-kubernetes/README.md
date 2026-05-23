# transpile-kubernetes

Resource DSL → Kubernetes Deployment manifest. Every function only
mutates context; the file template assembles the YAML.

```sh
../../capy run lib.yaml script.capy
```
