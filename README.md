<img src="./logo.svg" width="380">

**Note: Currently in Alpha**

**Kubernetes Global ConfigMap with namespace selector**

GonMap acts as the global configmap which creates children ConfigMaps
in the namespace matching the selector (or all namespaces if left empty)

# Installation
The [Helm chart](https://github.com/clobaa/gonmap-helm) is under construction. If you want to try it out,
just clone this repo and run
Note: below commands install CRDs in your current kube context. 
```sh
$ make install
$ make run
```

# Usage
```yaml
apiVersion: mondo.github.io.clobaa/v1
kind: GonMap
metadata:
  name: common-config
namespaceSelector:
  # To inject in some namespaces
  matchLabels:
    type: public
  # To inject in all the namespaces
  # matchLabels: {}
data:
  foo: bar
```