
https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration

### 编译
```bash
// 启用 go module
set GO111MODULE=on
// 为了方便从镜像里编译程序，将所有依赖都加载到vendor目录
go mod vendor
// 使用 docker 编译代码
imagesrcpath="/go/src/github.com/asdfsx/kubeconfig"
image="golang:1.11-alpine"
target="kubeconfig"
docker run --rm -v "${PWD}":"${imagesrcpath}" -w ${imagesrcpath} ${image} go build -v -o ${target}
// 生成镜像
docker build -t kubeconfig:latest .
```

### linter
安装`gometalinter`
```bash
brew tap alecthomas/homebrew-tap
brew install gometalinter
```
当前版本对`go module`支持还不好，所以只能在执行时关闭`go module`
```bash
go mod vendor
GO111MODULE=off gometalinter --skip=./vendor --deadline=5m
```


### 使用makefile编译
```bash
$ make help
  all                            build binary, then build and push image
  build                          build for docker
  clean                          Delete kubeconfig binary
  lint                           use gometalinter to check code
  push                           push image to docker registry
  vendor                         add dependencies to vendor directory
```

### 部署
```bash
kubectl create -f artifacts/init-namespace.yaml
kubectl create -f artifacts/kubeconfig-deploy.yaml
kubectl create -f artifacts/kubeconfig-ingress.yaml
```

### 本地测试运行
```bash
go build
NAMESPACE_PREFIX=clustar- ./kubeconfig -kubeconfig=~/.kube/hongkongconfig -swagger-ui-dist=./swagger-ui-dist/
```