# 📘 Kubebuilder BookAPI Controller

A Kubernetes controller built with [Kubebuilder](https://book.kubebuilder.io/) that manages a custom resource called `Kluster`. When a `Kluster` is created, the controller automatically provisions:

- A Deployment running a Book API application
- A NodePort Service to expose the API
- OwnerReferences so the Deployment and Service are deleted with the Kluster

## ⚙️ Prerequisites

- Go 1.22+
- Docker
- Kubernetes cluster (e.g., Minikube or kind)
- `kubectl` configured
- `make`

- ## 🚀 Install and Run

### 1. Install the CRD

```bash
make install
```

### 2. Deploy a Sample Kluster Resource

```bash
kubectl apply -f config/samples/core_v1alpha1_kluster.yaml

```


### 3. 🚀 Run the controller (in development mode)

```bash
make run
```
