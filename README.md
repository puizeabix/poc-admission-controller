# (Proof of Concept) Kubernetes custom admission controller

This is POC project to demostrate how to create sidecar injection admission controller, the goal is just to inject a simple `redis` container to any deployment in `default` namespace. This could be extended to inject any container depends on the use case

## Playground

1. Run `setup.sh` script to setup the certificate and server TLS keys
2. Run `kubectl apply -f manifest/deployment.yaml` to deploy admission controller
3. Run `kubectl apply -f manifest/admissionregistration.yaml` to register mutation review
4. Test by create deployment (in `default` namespace), e.g. `kubectl apply -f manifest.yaml` to create `nginx` deployment, the admission controller will inject `redis` as a sidecar