# Instructions to Run This App on GCE VM with Istio

1. Download setup script.

Download setup script.

```bash
curl -s -L https://raw.githubusercontent.com/incfly/istio-gce/master/install/setup.sh > setup.sh
```

1. Creat GKE cluster and install Istio with script.

```bash
export GCP_PROJECT="jianfeih-test"
export GCP_ZONE="us-central1-a"
export CLUSTER_NAME="microservice-demo"
export GCE_NAME="hipster-productcatalog"
bash ./setup.sh setup
```

1. Deploy microservice app in GKE cluster.

```bash
# skaffold run --default-repo=gcr.io/jianfeih-test -f 
# Install Istio CRD first.

```

Verify the app is up and running.

1. Delete productcatalogservice from Kubernetes clusters.

```bash
skaffold delete --default-repo=gcr.io/jianfeih-test -f istio-gce/skaffold-product.yaml
```

1. Modify `frontend.yaml` to point to VM instance IP and re-deploy.

1. Cleanup

```bash
bash ./setup.sh cleanup
```