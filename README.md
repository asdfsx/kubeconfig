
https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration

### 编译
```
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

### 直接运行
```
kubectl run --rm -i demo --image=asdfsx/kubeconfig --image-pull-policy=Always
```

### 部署
```
kubectl create -f artifacts/init-namespace.yaml
kubectl create -f artifacts/kubeconfig-deploy.yaml
kubectl create -f artifacts/kubeconfig-ingress.yaml
```
