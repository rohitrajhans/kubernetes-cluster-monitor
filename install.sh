# get current namespace

ns=$(kubectl config view --minify --output 'jsonpath={..namespace}')

# if no current context set, change current context to default

if [[ $ns == "" ]]; 
then    
	ns="default"; 
fi

# create new namespace: sidecar-injector

kubectl create ns sidecar-injector

# switch to the new namespace
kubectl config set-context --current --namespace=sidecar-injector

# first installing the webhook

kubectl create -f webhook/deployment/inject_sidecar.yaml

./webhook/deployment/webhook-create-signed-cert.sh \
--service sidecar-injector-webhook-svc \
--secret sidecar-injector-webhook-certs \
--namespace sidecar-injector

kubectl create -f webhook/deployment/clusterrole.yaml
kubectl create -f webhook/deployment/deployment.yaml
kubectl create -f webhook/deployment/service.yaml

kubectl create -f webhook/deployment/mutatingwebhook-ca-bundle.yaml
cat webhook/deployment/mutating-webhook.yaml | ./webhook/deployment/webhook-patch-ca-bundle.sh > webhook/deployment/mutatingwebhook-ca-bundle.yaml

# Setup the custom controller

kubectl create -f CustomResource/deployment/sample-operator.yaml

# Setup the logger

kubectl create -f logger/deployment/deployment.yaml

# setup for example
kubectl config set-context --current --namespace=default
kubectl create -f sidecar/clusterrole.yaml

# go back to initial namespace

kubectl config set-context --current --namespace=$ns

