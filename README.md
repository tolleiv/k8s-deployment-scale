# k8s-deployment-scale

This experiment provides a kubernetes client which allows scaling deployments through straight forward HTTP requests.
 
It's following the [go-client](https://github.com/kubernetes/client-go) library examples and not meant for anything else than testing purposes.

### Usage

The related docker image is hosted on [hub.docker.com](https://hub.docker.com/r/tolleiv/k8s-deployment-scale/). In order to use it, start a pod with the image, exposing port 8000 and provide a service.

### License

MIT License