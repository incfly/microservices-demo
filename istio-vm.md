# Instructions to Run This App on GCE VM with Istio

1.  Download setup script.

    ```bash
    wget https://raw.githubusercontent.com/incfly/istio-gce/master/install/istio-gce.sh
    ```

1.  Creat GKE cluster and install Istio with script.

    ```bash
    export GCP_PROJECT="jianfeih-test"
    export GCP_ZONE="us-central1-a"
    export GKE_NAME="microservice-demo"
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
    bash istio-gce.sh vm_exec sudo docker run -d  -p 3550:3550 gcr.io/jianfeih-test/productcatalogservice:2f7240f
    ```

1.  Delete productcatalogservice from Kubernetes clusters.

    ```bash
    skaffold delete --default-repo=gcr.io/jianfeih-test -f skaffold-product.yaml
    ```

    Now you should expect the product URL is not working.

1.  Register productcatalogservice running on GCE to the mesh.

    ```bash
    bash istio-gce.sh add_service productcatalogservice 3550 GRPC
    ```

1.  Wait for a while and then check product page, it works again!

## Migrate Redis to GCE

1. Change `redis.yaml` service port name to TCP (already done in this branch).

TODO: remove this workaround once https://github.com/istio/istio/issues/12139 is fixed.

1. Create a GCE instance and set up Redis service on VM instance.

    ```bash
    export GCE_NAME="hipster-rediscart"
    create_gce ${GCE_NAME}
    bash ./isti-gce.sh gce_setup
    # Install Docker manually
    gcloud compute ssh $GCE_NAME
    # Snippet from redis.yaml
    sudo docker run -d  -p  6379:6379  redis:alpine
    ```

1. Delete Kubernetes Service `redis-cart`. Expected output from frontend `httpHandlers error`.

    ```bash
    kubectl delete -f kubernetes-manifests/redis.yaml
    ```

1. Add redis service on VM to the mesh.

    ```bash
    # TODO: right now redis protocol does not work with Istio. We use TCP instead.
    bash istio-gce.sh add_service redis-cart 6379  TCP
    ```

1. Access frontend page, now the frontend page works again!

## Cleanup

    ```bash
    bash ./istio-gce.sh cleanup
    ```

## TODO Ideas

- Use tracing to debug the micro service bug? Even helpful for this demo itself setup.

## Debug Log

1. Redis with sidecar not working.
    Solution: mTLS disabled.
    Reason: I worked on Istio mutual TLS and MySQL issue recently...

1. How to disable sidecar injector.
   Solution: look up the istio.io documentation. Annotation with deployment not working. Ask ayj@ instead.

1. `ServiceEntry` redis-cart does not working.

1. Leftover `ServiceEntry` does affect redis service discovery.