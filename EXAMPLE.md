# Key point of usage
* client side must set the bootstrap content
  * suggest to use the env config `GRPC_XDS_BOOTSTRAP_CONFIG`
* The `target` of the client service must be `xds:///{k8s-service-name}.{service-namespace}:{port-number}`
  * Because the name of xDS `Listener` Resource is `targetHostPortNumber := net.JoinHostPort(fullName, strconv.Itoa(int(port.Port)))`
* The kubernetes Service Resource of the grpc server must have a name
  * Because if no name, the name of `ClusterLoadAssignment` Resource will be `portName = fmt.Sprintf("%s.%s:%d", ep.Name, ep.Namespace, port.Port)`
  * And it will miss-match the name of `Cluster` Resource, which is `targetHostPort := net.JoinHostPort(fullName, port.Name)`

## Client Example
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: dev
  name: greeter-client
  labels:
    app: greeter-grpc-client
spec:
  replicas: 1
  selector:
    matchLabels:
      app: greeter-grpc-client
  template:
    metadata:
      labels:
        app: greeter-grpc-client
    spec:
      containers:
        - name: greeter-grpc-client
          image: greeter-grpc-client:1.0.0 # use your own grpc client service image
          ports:
            - containerPort: 8080 
          env:
            - name: GRPC_GREET_SERVER_TARGET
              # must be xds:///{k8s-service-name}.{service-namespace}:{port-number}
              value: "xds:///service-greeter.dev:8972"
            - name: GRPC_XDS_BOOTSTRAP_CONFIG
              value: "{\"xds_servers\":[{\"server_uri\":\"xds-server.default:5000\",\"channel_creds\":[{\"type\":\"insecure\"}],\"server_features\":[\"xds_v3\"]}],\"node\":{\"id\":\"client-greeter\",\"locality\":{\"zone\":\"k8s\"}}}"
---
apiVersion: v1
kind: Service
metadata:
  namespace: dev
  name: client-greeter
spec:
  selector:
    app: greeter-grpc-client
  ports:
    - protocol: TCP
      port: 8080
      targetPort: 8080
```
## Server Example
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: dev
  name: greeter-service
  labels:
    app: greeter-grpc-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: greeter-grpc-service
  template:
    metadata:
      labels:
        app: greeter-grpc-service
    spec:
      containers:
        - name: greeter-grpc-service
          image: greeter-grpc-service:1.0.0 # use your own server image
          ports:
            - name: grpc
              containerPort: 8972
---
apiVersion: v1
kind: Service
metadata:
  namespace: dev
  name: service-greeter
spec:
  selector:
    app: greeter-grpc-service
  ports:
    - protocol: TCP
      # must have the port name
      name: grpc
      port: 8972
      targetPort: 'grpc'
```

