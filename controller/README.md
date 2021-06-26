
Setup the directory 

WSROOT=$(pwd)

mkdir -p $WSROOT/github.com

git clone https://github.com/naduvat/kubernetes-cluster-monitor.git

cd kubernetes-cluster-monitor/controller

MODULE=github.com/kubernetes-cluster-monitor/controller

GEN=<code generator path>

PACKAGE=k8s.crd.io:v1alpha1

Execute the generator

./tools/codegen.sh -g $GEN -m $MODULE -p $PACKAGE -w $WSROOT

make docker
