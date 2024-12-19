# Copyright 2024 Vega Cloud, Inc.

# Use of this software is governed by the Business Source License
# included in the file licenses/BSL.txt.

# As of the Change Date specified in that file, in accordance with
# the Business Source License, use of this software will be governed
# by the Apache License, Version 2.0, included in the file
# licenses/APL.txt.

curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
chmod +x ./kind
./kind create cluster --name testcluster-1 --config kind-config.yaml --image=kindest/node:v1.30.0
echo "Sleeping 20 seconds to let things settle down)
sleep(20)
kubectl apply -f ./test-resources.yaml
kubectl debug -n test-metrics test-pod --image=busybox -it --target=main-container
