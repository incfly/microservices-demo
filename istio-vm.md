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
    kubectl apply -f ./releases/istio-manifests.yaml

    # Deploy basic microservice app.
    kubectl apply -f release/kubernetes-manifests.yaml
    ```

    Verify the app is up and running.

1. Deploy productcatalog service in the created GCE instance.

    ```bash
    bash istio-gce.sh gce_setup 3550
    go build src/productcatalogservice/server.go
    gcloud compute scp server src/productcatalogservice/products.json $GCE_NAME:~
    # Run the productcatalog service without Docker.
    gcloud compute ssh ${GCE_NAME}
    ./server 2>&1 > product.log &
    ```

1. Delete productcatalogservice from Kubernetes clusters.

    ```bash
    kc delete svc/productcatalogservice  deployment/productcatalogservice
    ```

    Now you should expect the product URL is not working.

1.  Register productcatalogservice running on GCE to the mesh.

    ```bash
    bash istio-gce.sh add_service productcatalogservice 3550 GRPC
    ```

1.  Wait for a while and then check product page, it works again!

## Enable mTLS Policy

1. Now we try to enable mTLS between VM and Kubernetes workload.

    ```bash
    kubectl apply -f istio-gce/mtls.yaml
    ```

1. Open Grafana Dashboard, select the workload dashboard, and you will see the traffic changes from
   mTLS.

    ```bash
    kubectl -n istio-system port-forward $(kubectl -n istio-system get pod -l app=grafana -o jsonpath='{.items[0].metadata.name}') 3000:3000 &
    ```

## Advanced Routing Between VM and Kubernetes

Now we add back the Kubernetes services.

    ```bash
    # Redeploy Kubernetes serivces
    kubectl apply -f release/kubernetes-manifests.yaml
    python istio-gce/count.py
    ```

At the beginning, without `VirtualService` defined, Envoy randomly selects an endpoints from two
registries. The script prints out the result whether it hits VM or Kubernetes one.

<!-- TODO: consider to do it by default if you manages it via `istioctl` -->

Now let's simulate a traffic splitting to ramp up more on Kubernetes workload.

    ```bash
    kubectl apply -f istio-gce/product-dr-migration.yaml
    # Run the counting script again.
    python istio-gce/count.py
    ```

We should see the traffic have been shifting more to Kubernetes. We can even verify that in Grafana
service dashboard.

## VM Services Call Kubernetes Service

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