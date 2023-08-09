# xDS Server for gRPC on Kubernetes

A simple xDS server, distributing Kubernetes service endpoints to clients.

It is designed for [gRPC](https://grpc.github.io/grpc/cpp/md_doc_grpc_xds_features.html).

This software has been powering Wongnai and LINE MAN production since Q1 2022

## Why xDS?

Load balancing gRPC in Kubernetes is [notoriously complex problem](https://kubernetes.io/blog/2018/11/07/grpc-load-balancing-on-kubernetes-without-tears/).
Many solutions recommend using service mesh proxy to perform the load balancing instead.

With xDS support, we can now use gRPC client side load balancing with Kubernetes without writing per-language resolver.

## Why this xDS server

xDS has several features - traffic splitting, routing, retry, etc. Many xDS control plane like Istio or Crossover
implement those features with CRD.

The goal of this xDS server is to only solve gRPC load balancing. The design is simplistic:

1. It is self contained in one binary
2. There's no CRD to install
3. There's no external dependency and is fully stateless

## Usage

First you need to install this. Or you can just run this locally:

```shell
# Make sure you have go compiler installed
make
./.bin/k8sxds
```

This use your local kubeconfig, so if `kubectl` works then it should work, unless you use some sort of authentication
plugin.

If you're running in cluster it should also work without any environment variables. It requires some read-only
access, which you can find the ClusterRole in [deploy.yml](deploy.yml). It is recommended to deploy this as headless
service. As it use DNS-based discovery, we recommend not to use autoscaling on this service but keep it always at max
pods.

And that's it! Your xDS server is ready.

## Connecting to xDS from various languages
You'd need to set xDS bootstrap config on your application. Here's the xDS bootstrap file:

```json
{
    "xds_servers": [
        {
            "server_uri": "localhost:5000",
            "channel_creds": [{"type": "insecure"}],
            "server_features": ["xds_v3"]
        }
    ],
    "node": {
        "id": "anything",
        "locality": {
            "zone" : "k8s"
        }
    }
}
```

Make sure to change `server_url` to wherever your application can access this xDS server. You then can supply this to
your application in two methods:

- Put the entire JSON in `GRPC_XDS_BOOTSTRAP_CONFIG` environment variable
- Put the entire JSON in a file, then point `GRPC_XDS_BOOTSTRAP` environment variable to its path

Then follow the language specific instructions to enable xDS.

Finally, if you were connecting to `appname.appns:3000` write your connection string as `xds:///appname.appns:3000` instead.
Note the *triple* slash and that the namespace is *not* optional. (As this doesn't use DNS)

### Go

Add this somewhere, maybe in your main file

```go
import _ "google.golang.org/grpc/xds"
```

### Python

Make sure gRPC Python is at least v1.36. No code change is needed.

### JavaScript

Install [@grpc/grpc-js-xds](https://www.npmjs.com/package/@grpc/grpc-js-xds) 1.5.0 or later then run

```javascript
require('@grpc/grpc-js-xds').register();
```

Note that gRPC C (the [grpc](https://www.npmjs.com/package/grpc) package) is deprecated and does not contains xDS support.

### Java
You need to add grpc-xds dependency along with the common grpc dependencies.

```
<dependency>
	<groupId>io.grpc</groupId>
	<artifactId>grpc-netty</artifactId>
</dependency>
<dependency>
	<groupId>io.grpc</groupId>
	<artifactId>grpc-protobuf</artifactId>
</dependency>
<dependency>
    <groupId>io.grpc</groupId>
	<artifactId>grpc-services</artifactId>
</dependency>
<!-- xds protocol should work with this dependency -->
<dependency>
    <groupId>io.grpc</groupId>
	<artifactId>grpc-xds</artifactId>
	<scope>runtime</scope>
</dependency>
```

Then a new channel can be created with xds protocol.
```
Grpc.newChannelBuilder("xds:///{service}.{namespace}:{port}", InsecureChannelCredentials.create());
```

Note: the serviceConfigLookUp should not be disabled otherwise the xds protocol does not works correctly.

Since environment variable cannot be changed in java, there are 2 system properties which overrides the common bootstrap variables:- 
* io.grpc.xds.bootstrap to override GRPC_XDS_BOOTSTRAP
* io.grpc.xds.bootstrapConfig to override GRPC_XDS_BOOTSTRAP_CONFIG


## Virtual API Gateway

One feature of xDS is routing. This xDS server supports virtual API gateway by adding the following annotations to
Kubernetes service:

```yaml
apiVersion: v1
kind: Service
metadata:
  # ...
  annotations:
    xds.lmwn.com/api-gateway: apigw1,apigw2
    xds.lmwn.com/grpc-service: package.name.ExampleService,package.name.Example2Service
```

The service also must have a port named `grpc`, which traffic will be sent to.

Then client applications (with xDS support) can connect to `xds:///apigw1` or `xds:///apigw2` (no port or namespace)
and any API calls to gRPC service `package.name.ExampleService` and `package.name.Example2Service` will be sent to this
service.

## Scalability

Since xDS is also gRPC based, it might beg the question "how do we load balance the load balancer"?

As xDS is only the control plane, the load on xDS itself should be really light - it only contains watches for all
possible service pairs, and data for all Kubernetes endpoints. Therefore it should be possible to prescale xDS to the
desired capacity and use DNS to discover xDS control plane.

The question remains that how many service pairs this would be able to support, and how large a Kubernetes cluster it
is able to support. At Wongnai we run a cluster with hundreds of services, and this service are able to handle all data
for all namespaces just fine.

Finally, as xDS is only the control plane, in case of outages any new/removed endpoints will not be known by clients
but existing connections will remain flowing. gRPC automatically reconnects to xDS control plane in this case. 

## License

© 2022 Wongnai Media Co, Ltd.

This software is licensed under [MIT License](LICENSE)
