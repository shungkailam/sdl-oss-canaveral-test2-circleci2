#!/bin/bash

function create_devtools_redis_cluster {
    # Create devtools svc for few namespaces only
    echo "Checking if redis cluster present..."
    clusterStatusGet=`kubectl exec -it devtools-redis-statefulset-0 -n $NS_NAME -- redis-cli cluster info | awk 'NR==1{print $1}'`
    clusterStatus="${clusterStatusGet//$'\r'/}"
    clusterStatusOK="cluster_state:ok"
    if [ "$clusterStatus" = "$clusterStatusOK" ]; then
        echo "cluster is already formed"
    else
        echo "Creating redis cluster..."
        echo 'yes' | kubectl exec -it devtools-redis-statefulset-0 -n $NS_NAME -- redis-cli --cluster create --cluster-replicas 1 $(kubectl get pods -l app=devtools-redis-cluster -n $NS_NAME -o jsonpath='{range.items[*]}{.status.podIP}:6379 ')
        # Above command asks for user input if he wants to accept cluster configuration
    fi
}

echo "Namespace: $NS_NAME"
if [ "$NS_NAME" = "beta" ] || [ "$NS_NAME" = "stage" ] || [ "$NS_NAME" = "go" ] || [ "$NS_NAME" = "test" ] || [ "$NS_NAME" = "uie2e" ] || [ "$NS_NAME" = "multinode" ] || [ "$NS_NAME" = "my" ] ; then
    while true; do 
        redisPodCount=`kubectl get pod -n $NS_NAME | grep "devtools-redis-statefulset" | grep "Running"  | wc -l`
        redisStatus="${redisPodCount//$'\r'/}"
        echo "RedisStatus: ${redisStatus}"
        if [ "$redisStatus" = "$REDIS_POD_COUNT" ]; then
            create_devtools_redis_cluster
            break
        else
            echo "Waiting for $REDIS_POD_COUNT redis pods: $redisPodCount are ready"
            sleep 5
        fi
    done
else
    echo "Not creating redis cluster as current namespace is not set to have cluster"
fi