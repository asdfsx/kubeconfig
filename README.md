
https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration

### 编译
```bash
// 启用 go module
set GO111MODULE=on
// 为了方便从镜像里编译程序，将所有依赖都加载到vendor目录
go mod vendor
// 使用 docker 编译代码
imagesrcpath="/go/src/github.com/starcloud-ai/kubeconfig"
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
kubectl create -f artifacts/kubeconfig-namespace.yaml
kubectl create -f artifacts/kubeconfig-deploy.yaml
kubectl create -f artifacts/kubeconfig-ingress.yaml
```

### 本地测试运行
```bash
go build
NAMESPACE_PREFIX=clustar- ./kubeconfig -kubeconfig=~/.kube/hongkongconfig -swagger-ui-dist=./swagger-ui-dist/
```

### 测试rbac
首先通过swagger-ui创建测试空间`clustar-sample`，测试用户`sample`。获得kubeconfig，并保存到本地。

使用获得到kubeconfig执行下边的命令。修改namespace，看看是否报错，`clustar-sample`之外应该只有只读权限。

```text
# create pod
cat <<EOF|kubectl --kubeconfig=./testconfig create -f -
apiVersion: v1
kind: Pod
metadata:
  name: rbac-test
  namespace: clustar-sample
spec:  # specification of the pod's contents
  containers:
  - name: rbac-test
    image: "busybox"
    command: ["top"]
    stdin: true
    tty: true
EOF

# get pod
kubectl --kubeconfig=./testconfig get pod --all-namespaces

# delete pod
kubectl --kubeconfig=./testconfig delete pod rbac-test -n clustar-sample

# create ServiceAccount
cat <<EOF|kubectl --kubeconfig=./testconfig create -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sample2
  namespace: clustar-sample
EOF

# delete ServiceAccount
cat <<EOF|kubectl --kubeconfig=./testconfig delete -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sample2
  namespace: clustar-sample
EOF

```


### 测试arena
1. 下载arena
```bash
mkdir /charts
git clone https://github.com/kubeflow/arena.git
cp -r arena/charts/* /charts
```

2. 安装
```bash
kubectl create -f arena/kubernetes-artifacts/jobmon/jobmon-role.yaml
kubectl create -f arena/kubernetes-artifacts/tf-operator/tf-operator.yaml
kubectl create -f arena/kubernetes-artifacts/dashboard/dashboard.yaml
```

3. 创建一个单独的namespace
```bash
curl -X POST "http://kubeconfig.hongkong.ai/kubeconfig" \
     -H "accept: application/json" \
     -H "Content-Type: application/json" \
     -d "{ \"namespace\": \"clustar-sample\", \"serviceaccount\": \"sample\"}"
```

4. 获取kubeconfig
```bash
curl http://kubeconfig.hongkong.ai/kubeconfig/clustar-sample/sample > testconfig 
```

5. 使用arena提交任务到新创建到namespace中
```sh
arena submit tf \
             --config=testconfig \
             --namespace=clustar-sample \
             --name=tf-git \
             --image=tensorflow/tensorflow:1.11.0 \
             --syncMode=git \
             --syncSource=https://github.com/cheyang/tensorflow-sample-code.git \
             --loglevel=debug \
             "python code/tensorflow-sample-code/tfjob/docker/mnist/main.py --max_steps 100"
             
```