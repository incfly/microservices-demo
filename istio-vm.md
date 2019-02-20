# Instructions to Run This App on GCE VM with Istio

1.  Download setup script.

    ```bash
    wget https://raw.githubusercontent.com/incfly/istio-gce/master/install/istio-gce.sh
    ```

1.  Creat GKE cluster and install Istio with script.

    ```bash
    export GCP_PROJECT="jianfeih-test"
    export GCP_ZONE="us-central1-a"
    export CLUSTER_NAME="microservice-demo"
    export GCE_NAME="hipster-productcatalog"
    bash ./istio-gce.sh setup
    ```

    This will create a GKE cluster, a GCE instance, and install Istio control
    plane in the GKE cluster.

1.  Deploy microservice app in GKE cluster.

    ```bash
    # Install Istio CRD first.
    kubectl apply -f ./istio-manifests

    # Deploy basic microservice app.
    skaffold run --default-repo=gcr.io/jianfeih-test -f skaffold-istio-gce.yaml
    ```

    Verify the app is up and running.

1.  Deploy productcatalog service in the created GCE instance.

    <!-- TODO: WIP, here, consolidate gce-vm.sh script and setup GCE instance. -->

    ```bash
    # You may need to install Docker manuall with `gcloud compute ssh`
    bash istio-gce.sh gce_setup
    # Run the Docker with productservice
    bash istio-gce.sh vm_exec docker run -d  -p 3550:3550 gcr.io/jianfeih-test/productcatalogservice:2f7240f
    ```

1.  Delete productcatalogservice from Kubernetes clusters.

    ```bash
    skaffold delete --default-repo=gcr.io/jianfeih-test -f istio-gce/skaffold-product.yaml
    ```

    Now you should expect the product URL is not working.

1.  Register productcatalogservice running on GCE to the mesh.

    ```bash
    bash istio-gce.sh add_service productcatalogservice 3550 GRPC
    ```

1.  Wait for a while and then check product page, it works again!

1.  Cleanup

    ```bash
    bash ./istio-gce.sh cleanup
    ```
