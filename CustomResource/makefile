reset:
	kubectl delete -f deployment/sample-operator.yaml

restart:
	kubectl delete -f deployment/sample-operator.yaml &&\
	kubectl apply -f deployment/sample-operator.yaml &&\
	kubectl apply -f deployment/sample-resource.yaml