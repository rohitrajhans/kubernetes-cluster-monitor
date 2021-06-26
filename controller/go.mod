module github.com/kubernetes-cluster-monitor/controller

go 1.15

require (
	github.com/mattbaird/jsonpatch v0.0.0-20200820163806-098863c1fc24
	github.com/sirupsen/logrus v1.8.1
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.0.0-00010101000000-000000000000
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20210313030403-f6ce18ae578c
