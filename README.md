Dynamic Configuration Operator
==============================

Kubernetes Operator that automatically update deployment when an upstream ConfigMap or Secret is updated. 
This operator allows updating configuration for apps that:
* Reads configuration during startup and does not have a live-reload feature.
* Uses [`subPath`](https://kubernetes.io/docs/concepts/storage/volumes/#using-subpath) while mounting a ConfigMap or Secret.
* Uses [Projected Volumes](https://kubernetes.io/docs/concepts/storage/projected-volumes/).

Built with Go and Operator SDK.

## Tests

Execute unit test with Make:

```
$ make test
```

## Deploy

```
$ make deploy
```

## Usage

Given an existing Deployment using a ConfigMap, label both resources with `app.lebiller.dev/dynamic-configuration=watch`:

```
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx
  labels:
    app.lebiller.dev/dynamic-configuration: watch
data:
  index.html: hello-world
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  labels:
    app.lebiller.dev/dynamic-configuration: watch
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
          volumeMounts:
            - name: nginx
              mountPath: /usr/share/nginx/html/index.html
              subPath: index.html
      volumes:
        - name: nginx
          configMap:
            name: nginx
```

After the first deployment, every change happening on the ConfigMap will be detected by the operator 
and the deployment's template annotation `app.lebiller.dev/configuration-hash` will be updated,
effectively triggering a new deployment rollout.
