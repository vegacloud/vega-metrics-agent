curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
chmod +x ./kind
./kind create cluster --name test-cluster --config kind-config.yaml --image=kindest/node:v1.30.0
echo "Sleeping 20 seconds to let things settle down)
sleep(20)
kubectl apply -f ./test-resources.yaml
