## Create 3-node environment config
kind create cluster --config config.yaml

## Create cluster
kind kind create cluster --config config.yaml
Creating cluster "multustesting" ...
 ✓ Ensuring node image (kindest/node:v1.25.3) 🖼
 ✓ Preparing nodes 📦 📦 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
 ✓ Joining worker nodes 🚜 
Set kubectl context to "kind-multustesting"
You can now use your cluster with:

kubectl cluster-info --context kind-multustesting

Have a question, bug, or feature request? Let us know! https://kind.sigs.k8s.io/#community 🙂

## get node
kind kubectl get nodes        
NAME                          STATUS   ROLES           AGE     VERSION
multustesting-control-plane   Ready    control-plane   3m4s    v1.25.3
multustesting-worker          Ready    <none>          2m40s   v1.25.3
multustesting-worker2         Ready    <none>          2m40s   v1.25.3

## install multus
kubectl create -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset.yml
customresourcedefinition.apiextensions.k8s.io/network-attachment-definitions.k8s.cni.cncf.io created
clusterrole.rbac.authorization.k8s.io/multus created
clusterrolebinding.rbac.authorization.k8s.io/multus created
serviceaccount/multus created
configmap/multus-cni-config created
daemonset.apps/kube-multus-ds created

## get koko
curl -LO https://github.com/redhat-nfvpe/koko/releases/download/v0.82/koko_0.82_linux_amd64
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   611    0   611    0     0   1253      0 --:--:-- --:--:-- --:--:--  1254
100 14.6M  100 14.6M    0     0  2652k      0  0:00:05  0:00:05 --:--:-- 3233k
[tohayash@tohayash-lab kind]$ chmod +x koko_0.82_linux_amd64 

## Create veth interface between kind-woker and kind-worker2
sudo ./koko_0.82_linux_amd64 -d multustesting-worker,eth1 -d multustesting-worker2,eth1
Create veth...done

## install CNI reference plugin from github
kubectl create -f cni-install.yaml

## create network attatched definition
➜  kind kubectl apply -f nattachdef.yaml
networkattachmentdefinition.k8s.cni.cncf.io/centos-runtimeconfig-def created
## Create pods for the network attached definition
➜  kind kubectl apply -f multuspods.yaml 
pod/centos-worker1 created
pod/centos-worker2 created

## Something broke on the nodes beats me what did but needed to do this
sysctl net/netfilter/nf_conntrack_max=131072

Otherwise, kube-proxy keeps blowing up on the worker nodes.
https://github.com/kubernetes-sigs/kind/issues/2240