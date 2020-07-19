standalone-etcd:
	docker run \
	-e ALLOW_NONE_AUTHENTICATION=yes \
    -e ETCD_ADVERTISE_CLIENT_URLS=http://localhost:2379 \
	-p 2379:2379 \
	-p 2380:2380 \
	bitnami/etcd:latest