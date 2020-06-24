
# Kubernetes Policy-based Framework for Cluster Failure Prevention

## Description:
The framework provides an easy-to-use solution to implement policies to target a group of containers. An independent analytics tool capable of identifying incoming threats to the cluster is part of the framework. The framework can quarantine the pods to prevent the spread of the vulnerability that can compromise the entire service. User ease has been a top priority for the framework, which provides easy installation and a familiar approach to apply policies to the Kubernetes cluster.

## Aim:
- Easy cluster-wide setup
- Intuitive policy configuration
- Independent analytics tool
- Preventive actions to avoid failures


## Use Cases:
- The framework deploys a sidecar to monitor the health of the main application container in the pod. The data collected by the sidecar can be used by various analytical engines. The analysis would be beneficial in manging the cluster health and to avoid failures.
- In case a pod on a worker node fails, it can be quarantined and possibly pushed to another node. A service disruption can be prevented by taking appropriate action on similar pods.

## Architecture Overview:
![Architecture](https://raw.githubusercontent.com/rohitrajhans/kubernetes-cluster-monitor/master/media/framework.png)

## Installation:
- #### Requirements:
    - Access to a running Kubernetes cluster and kubectl
    - On a local system, it can be run by using [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)
    - Docker (using version 19.03.8), Go (using Version 1.14)
    - Ensure that a docker image has been created for all components (steps mentioned below)
- #### Easy framework setup
	1. Run `./install.sh` : This will install all the required components
	2. Add label `sidecar-injection: enabled` to the namespace where you want to add sidecar
	3. Run `kubectl create -f sidecar/clusterrole.yaml` to apply necessary RBAC permissions to access the namespace where you want to add the sidecar. Currently `sidecar/clusterrole.yaml` adds necessary permissions for `namespace: default`. This can be edited to apply to any other namespace.
	4. Create a policy. A sample policy can be observed in `CustomResource/deployment/sample-resource.yaml`. Apply the policy to the cluster using `kubectl create -f CustomResource/deployment/sample-resource.yaml`.
	5. The setup is now complete. A sidecar will be injected to all the matching pods.

## Individual Component Setup:
- Before setting up framework, create **docker images** for each individual component:
		1. Custom Resource: Run `CustomResource/build.sh`
		2. Wehook: Run `cd webhook; make`
		3. Log Aggregator: Run `cd logger;  make build-collector`
		4. Log API: Run `cd logger; make build-endpoint`
		5. Log Sender: Run `make build-sender`
		6. Sidecar: Run `cd sidecar; make`
		
- Setting up the **Mutating Admission Controller** for sidecar injection:
	1. Switch context to `namespace: sidecar-injector` 
	2. Run `kubectl create -f webhook/deployment/inject_sidecar.yaml` to create sidecar configuration
	3. Run `./webhook/deployment/webhook-create-signed-cert.sh \
--service sidecar-injector-webhook-svc \
--secret sidecar-injector-webhook-certs \
--namespace sidecar-injector` to create necessary certificates for enabling TLS communication with kube-apiserver
	4. Run `kubectl create -f webhook/deployment/clusterrole.yaml` for necessary RBAC permissions 
	5. Run` kubectl create -f webhook/deployment/deployment.yaml &&
kubectl create -f webhook/deployment/service.yaml`
	6. Run `kubectl create -f webhook/deployment/mutatingwebhook-ca-bundle.yaml &&
cat webhook/deployment/mutating-webhook.yaml | ./webhook/deployment/webhook-patch-ca-bundle.sh > webhook/deployment/mutatingwebhook-ca-bundle.yaml` 

- Setting up the Custom Resource:
	1. Switch context to `namespace: sidecar-injector`
	2. Run `kubectl create -f CustomResource/deployment/sample-operator.yaml` to setup the custom controller.

- Setting up the Custom Resource:
	1. Switch context to `namespace: sidecar-injector`
	2. Run `kubectl create -f logger/deployment/deployment.yaml` to setup the logger application.