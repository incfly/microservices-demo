# Instructions to Run This App on GCE VM with Istio

1. Download setup script.

Download setup script.

```bash
wget https://raw.githubusercontent.com/incfly/istio-gce/master/install/istio-gce.sh
```

1. Creat GKE cluster and install Istio with script.

```bash
export GCP_PROJECT="jianfeih-test"
export GCP_ZONE="us-central1-a"
export CLUSTER_NAME="microservice-demo"
export GCE_NAME="hipster-productcatalog"
bash ./istio-gce.sh setup
```

This will create a GKE cluster, a GCE instance, and install Istio control plane in the GKE cluster.

1. Deploy microservice app in GKE cluster.

```bash
# Install Istio CRD first.
kubectl apply -f ./istio-manifests

# Deploy basic microservice app.
skaffold run --default-repo=gcr.io/jianfeih-test -f skaffold-istio-gce.yaml
```

Verify the app is up and running.

1. Deploy productcatalog service in the created GCE instance.

<!-- TODO: WIP, here, consolidate gce-vm.sh script and setup GCE instance. -->

```bash
bash istio-gce.sh gce_setup
```

1. Delete productcatalogservice from Kubernetes clusters.

```bash
skaffold delete --default-repo=gcr.io/jianfeih-test -f istio-gce/skaffold-product.yaml
```

1. Modify `frontend.yaml` to point to VM instance IP and re-deploy.

1. Cleanup

```bash
bash ./setup.sh cleanup
```