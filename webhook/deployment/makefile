bundle:
	cat mutating-webhook.yaml | ./webhook-patch-ca-bundle.sh > mutatingwebhook-ca-bundle.yaml

context-default:
	kubectl config set-context --current --namespace=default

context-sidecar:
	kubectl config set-context --current --namespace=sidecar-injector

context:
	kubectl config set-context --current --namespace=$(ns)

reset:
	kubectl delete -f mutatingwebhook-ca-bundle.yaml &&\
	kubectl delete -f service.yaml &&\
	kubectl delete -f deployment.yaml &&\
	kubectl delete -f clusterrole.yaml &&\
	kubectl delete -f inject_sidecar.yaml

apply-config:
	kubectl create -f inject_sidecar.yaml

deploy:
	kubectl create -f clusterrole.yaml &&\
	kubectl create -f deployment.yaml &&\
	kubectl create -f service.yaml

secret:
	./webhook-create-signed-cert.sh \
    --service sidecar-injector-webhook-svc \
    --secret sidecar-injector-webhook-certs \
    --namespace sidecar-injector
