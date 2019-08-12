// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/productcatalogservice/genproto"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"cloud.google.com/go/profiler"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/golang/protobuf/jsonpb"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

var (
	cat          pb.ListProductsResponse
	catalogMutex *sync.Mutex
	log          *logrus.Logger
	extraLatency time.Duration

	port = "3550"

	reloadCatalog bool
)

func init() {
	log = logrus.New()
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
	catalogMutex = &sync.Mutex{}
	err := readCatalogFile(&cat)
	if err != nil {
		log.Warnf("could not parse product catalog")
	}
}

func main() {
	go initTracing()
	go initProfiling("productcatalogservice", "1.0.0")
	flag.Parse()

	// set injected latency
	if s := os.Getenv("EXTRA_LATENCY"); s != "" {
		v, err := time.ParseDuration(s)
		if err != nil {
			log.Fatalf("failed to parse EXTRA_LATENCY (%s) as time.Duration: %+v", v, err)
		}
		extraLatency = v
		log.Infof("extra latency enabled (duration: %v)", extraLatency)
	} else {
		extraLatency = time.Duration(0)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for {
			sig := <-sigs
			log.Printf("Received signal: %s", sig)
			if sig == syscall.SIGUSR1 {
				reloadCatalog = true
				log.Infof("Enable catalog reloading")
			} else {
				reloadCatalog = false
				log.Infof("Disable catalog reloading")
			}
		}
	}()

	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Infof("starting grpc server at :%s", port)
	run(port)
	select {}
}

func run(port string) string {
	l, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}
	cert, err := tls.X509KeyPair(
		[]byte(`-----BEGIN CERTIFICATE-----
MIICvDCCAaQCCQDQWsB0wB38PTANBgkqhkiG9w0BAQsFADAgMR4wHAYDVQQDDBVw
cm9kdWN0Y2F0YWxvZ3NlcnZpY2UwHhcNMTkwODEyMjIyNjE2WhcNMjAwODExMjIy
NjE2WjAgMR4wHAYDVQQDDBVwcm9kdWN0Y2F0YWxvZ3NlcnZpY2UwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQCvmdIWLQ6biFqDGORQtu7ubw+TvIt+rTwF
qFkVxsq1fs/nLrM6nK3RbbiM7qUQJFjCPCld/I2o6ZICivlxq2WE6dlA9q5WhOgR
/yzze0pLHlS+qET7NqWCFSayPg6tzDPbfAA/NJhfpIcg160adeH+sX1Ns5BuPSpr
u4Q+cw4bScEIbjXWzfl1nvy70AUeOTqJpRA6k84625JEWy7++pla0Bl6m1xwfHqz
SXAw5raSNEdg/g6454t1FC8I0sTVmKELOdmtgnq02qQb1ktHgHlWUu5n+Xecbo6D
ELOOxEm1RrffxwduqhzWW9yg5lu9KikI4T1nhqywkkczyn+N7WApAgMBAAEwDQYJ
KoZIhvcNAQELBQADggEBAEyEY9XAk8cDGu8YqY5kt+4BXxzZMOqe3ZnyRywvLLIK
6XZJ+UPWiW71ZO5Ow50ji+LIPgqU6mFN0TFJfwZ3Qz8GabtOJpwTDkWbVDAYZK8o
RTT2lH/+5MmpywglegwAjtYHLXREcbqTe3MvHoUTWrrilTMRPmkVToeqAwdupVWD
0KFJ8par+ZA+hFii6DSGeGwymZwMHM3x7BfWxQ2uxmZd0C5KF40MJJQlN77+9n9K
HCPO65OM/QGt6oaORKKuby9+2xlrQ1b+ufKFq2fsduDYIM7JhlhFIvueIF9Zb+2p
rz2G05xLSPoiZDBLQASMUGTQcHO2D6clPl+/O7QbGzU=
-----END CERTIFICATE-----`),
		[]byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAxHav4NFewEKRmrRz+lCDvvRFMA4sj2gM08AGSbcHwBULgjHd
jBETljKpecq1dFyv2w31G3tWsFP5NFD+IlMUx3DbDUdXiWOV7ynHv8CUnCRAgbWs
Df1LDtRhZoAg+fk52KQ27OE6a8tjXgIfE/Uue99b7xeK0977KyR0NnUTocReA189
zuI4YI1ytbQC25mAfIvQMlcqqW52KnXxlMLaHB6v8xdLhOTwrL363cP/f7h+R4nW
qwu08vsmEaSWwk4wF7hSAjw+WMffv7OANSz2XWip6+UDUDcWoCQwPsTJdnOhd8Tw
o9yb1kw3T1Ey9gVo9suO/7kWwN1MC2GF5p4TDwIDAQABAoIBADtU3Ki4kjTd5bsi
5COkTSVN/9cTcMGeWsFYLI32iJCpyl/3T0ENpyylACmX3lTV6QXuoSc7iGKX+Zqj
GxyimpPgsUbBVN5ZBN7Fb12pezfses6xXtSauiAwY3nhGBRl/+I9NZk0K8CCG/A4
E8qjMPaX7du28GYr4Q6WY8JOeS9Q5iNSnmnOHpKkRNJlWO3b292NxOTFh4OQNgEI
uJj5ce68RdqS4Fj14Jt4gtwqmGkqCUJJ+diqX1uUw+2fj6Hf+f3GznYObDvc7o2n
dBf0zRys5y41UbiQD+/eLTbMs6A8ll2cS9GVEpas0ehQ4J7Yc6oX5bR3N4ZvlEN/
vOkHdoECgYEAyiOwwugEg9xIUWrtRI8+tfQaKd5IFe1o/uw+SpmEqb2u3vQRECwn
ZGmY2BMfAMXuCwP0rDb2hXxjvdaWXeKUFA7J8r+lw8GvnHHrJ4/A3z69hxR1rvoy
F7exPbNsZHutUcVMN5o2doJ7FefQNQDXSf/zk4t5kTLH5BwJkhETV6MCgYEA+M/X
BTE07j8ceIot3baqjIhRv1YzWH9IzvmGlV5YvqVx05kXRIGi7ysFSW5gb0JZAlDJ
KS11G4TRXDJRhGSYbgReLM3oMgYcD23IUtvxTVJ75q6A69o9LUQnPZqIEMojWr2c
0UdPLAFAJyHuj6fenmn3cQzP0b5rJx56GCT0faUCgYA6OS+H5IawaHnYIcF39v6s
MER8/M6sqjaM/wUuPavtrHo7M/faPa2XCaeBzXgno9teBuSp2icF6f9cxfuHzWSz
plLa/gLEMPzhRhriyVBXvV2gE++V1/EnzbxatlypUMpqfDbo6R144zqK47ugGL7q
TLQfMpRwkzzqYn0LOqnkmwKBgD8NbJAESEWX+L8TRUxKXi3+3bh/P8PNfcX1tgVk
Q1kM1CurQBo8P+4cGNri/c00IxpTHqcwvdyba/LRTZcfZwF6WeNAyvbiVXoTeBCH
bD8MCBoNXt5mD9rIyqjx4Elg8FSueG8Qgx/DsV45WxtMjz3V3L7pYEDm4ICpWIeF
1e+BAoGADkkFlsHu4IkOcz1Uqh3yWjorMhEUurdSjXxrEz2soKjzHILqg0GQ+nju
+vsjVslm79Uh+9jRZKDUX/tR7FNbvR1RZZbpvaBO+Xnpm8jxf84UzZYaodgiPUHo
CEQAnPTZ42NtvFXDeKfN6cyEJ0XIElJExQteJsWd+Q2+RxTubpk=
-----END RSA PRIVATE KEY-----`))
	if err != nil {
		log.Fatal(err)
	}
	srv := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)))
	svc := &productCatalog{}
	pb.RegisterProductCatalogServiceServer(srv, svc)
	healthpb.RegisterHealthServer(srv, svc)
	go srv.Serve(l)
	return l.Addr().String()
}

func initJaegerTracing() {
	svcAddr := os.Getenv("JAEGER_SERVICE_ADDR")
	if svcAddr == "" {
		log.Info("jaeger initialization disabled.")
		return
	}
	// Register the Jaeger exporter to be able to retrieve
	// the collected spans.
	exporter, err := jaeger.NewExporter(jaeger.Options{
		Endpoint: fmt.Sprintf("http://%s", svcAddr),
		Process: jaeger.Process{
			ServiceName: "productcatalogservice",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	trace.RegisterExporter(exporter)
	log.Info("jaeger initialization completed.")
}

func initStats(exporter *stackdriver.Exporter) {
	view.SetReportingPeriod(60 * time.Second)
	view.RegisterExporter(exporter)
	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		log.Info("Error registering default server views")
	} else {
		log.Info("Registered default server views")
	}
}

func initStackdriverTracing() {
	// TODO(ahmetb) this method is duplicated in other microservices using Go
	// since they are not sharing packages.
	for i := 1; i <= 3; i++ {
		exporter, err := stackdriver.NewExporter(stackdriver.Options{})
		if err != nil {
			log.Warnf("failed to initialize Stackdriver exporter: %+v", err)
		} else {
			trace.RegisterExporter(exporter)
			trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
			log.Info("registered Stackdriver tracing")

			// Register the views to collect server stats.
			initStats(exporter)
			return
		}
		d := time.Second * 10 * time.Duration(i)
		log.Infof("sleeping %v to retry initializing Stackdriver exporter", d)
		time.Sleep(d)
	}
	log.Warn("could not initialize Stackdriver exporter after retrying, giving up")
}

func initTracing() {
	initJaegerTracing()
	initStackdriverTracing()
}

func initProfiling(service, version string) {
	// TODO(ahmetb) this method is duplicated in other microservices using Go
	// since they are not sharing packages.
	for i := 1; i <= 3; i++ {
		if err := profiler.Start(profiler.Config{
			Service:        service,
			ServiceVersion: version,
			// ProjectID must be set if not running on GCP.
			// ProjectID: "my-project",
		}); err != nil {
			log.Warnf("failed to start profiler: %+v", err)
		} else {
			log.Info("started Stackdriver profiler")
			return
		}
		d := time.Second * 10 * time.Duration(i)
		log.Infof("sleeping %v to retry initializing Stackdriver profiler", d)
		time.Sleep(d)
	}
	log.Warn("could not initialize Stackdriver profiler after retrying, giving up")
}

type productCatalog struct{}

func readCatalogFile(catalog *pb.ListProductsResponse) error {
	catalogMutex.Lock()
	defer catalogMutex.Unlock()
	catalogJSON, err := ioutil.ReadFile("products.json")
	if err != nil {
		log.Fatalf("failed to open product catalog json file: %v", err)
		return err
	}
	if err := jsonpb.Unmarshal(bytes.NewReader(catalogJSON), catalog); err != nil {
		log.Warnf("failed to parse the catalog JSON: %v", err)
		return err
	}
	log.Info("successfully parsed product catalog json")
	return nil
}

func parseCatalog() []*pb.Product {
	if reloadCatalog || len(cat.Products) == 0 {
		err := readCatalogFile(&cat)
		if err != nil {
			return []*pb.Product{}
		}
	}
	return cat.Products
}

func (p *productCatalog) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (p *productCatalog) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (p *productCatalog) ListProducts(context.Context, *pb.Empty) (*pb.ListProductsResponse, error) {
	time.Sleep(extraLatency)
	return &pb.ListProductsResponse{Products: parseCatalog()}, nil
}

func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	time.Sleep(extraLatency)
	var found *pb.Product
	for i := 0; i < len(parseCatalog()); i++ {
		if req.Id == parseCatalog()[i].Id {
			found = parseCatalog()[i]
		}
	}
	if found == nil {
		return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
	}
	return found, nil
}

func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	time.Sleep(extraLatency)
	// Intepret query as a substring match in name or description.
	var ps []*pb.Product
	for _, p := range parseCatalog() {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(req.Query)) ||
			strings.Contains(strings.ToLower(p.Description), strings.ToLower(req.Query)) {
			ps = append(ps, p)
		}
	}
	return &pb.SearchProductsResponse{Results: ps}, nil
}
