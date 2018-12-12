# Containerized tuned daemon for OpenShift

This is a proof-of-concept for running the tuned daemon inside a pod on OCP.

## Sample run

If you do not wish to build a containerized tuned image yourself, you can skip
the following section and use a prebuilt `docker.io/jmencak/tuned-poc` image.

### Build a docker image for the containerized tuned

First build a docker image for the containerized tuned.

```
(cd image && docker build -t my-tuned-image .)
```

Push the image to a public registry.

### Deploy containerized tuned on the OCP cluster

Create the objects on the OCP cluster 

```
for f in objects/*.yaml ; do oc create -f $f ; done
```

### View and change the node-level profiles

```
$ oc project tuned
Now using project "tuned" on server "https://b4.lan:8443".
$ oc get cm
NAME              DATA      AGE
tuned-profiles    1         1h
tuned-recommend   1         1h
```

`tuned-profiles` and `tuned-recommend` are ConfigMaps that the OCP administrator can 
use to set the node-level tuning on the OCP cluster.  `tuned-profiles` contains all profiles
for the OCP cluster and `tuned-recommend` is a `recommend.conf` file to set the recommended profiles
for a node based on node labels and potentially other attributes the tuned recommend functionality
allows.

The tuned pod uses inotify events to catch ConfigMap changes and reloads the profiles based on
a newly recommended profile.  Node label changes are listened to by using a pull model by querying
OpenShift API server to fetch node labels.

```
user@b1: ~ $ oc get nodes --show-labels
NAME      STATUS    ROLES     AGE       VERSION           LABELS
b1.lan    Ready     master    1h        v1.10.0+b81c8f8   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/hostname=b1.lan,node-role.kubernetes.io/master=true
b2.lan    Ready     infra     1h        v1.10.0+b81c8f8   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/hostname=b2.lan,node-role.kubernetes.io/infra=true,role=node
b3.lan    Ready     compute   1h        v1.10.0+b81c8f8   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/hostname=b3.lan,node-role.kubernetes.io/compute=true,role=node
```

Let's investigate the profiles currently set on the nodes.

```
user@b1: ~ $ oc get pods -o wide
NAME          READY     STATUS    RESTARTS   AGE       IP             NODE
tuned-l7q5b   1/1       Running   0          1h        172.16.113.3   b3.lan
tuned-wmvkq   1/1       Running   0          1h        172.16.113.2   b2.lan
tuned-zv6h9   1/1       Running   0          1h        172.16.113.1   b1.lan
user@b1: ~ $ oc logs tuned-zv6h9 | grep applied
2018-08-14 10:45:37,176 INFO     tuned.daemon.daemon: static tuning from profile 'openshift-control-plane' applied
user@b1: ~ $ oc logs tuned-wmvkq | grep applied
2018-08-14 10:45:36,161 INFO     tuned.daemon.daemon: static tuning from profile 'openshift-control-plane' applied
user@b1: ~ # oc logs tuned-l7q5b | grep applied
2018-08-14 10:45:36,483 INFO     tuned.daemon.daemon: static tuning from profile 'openshift-node' applied
```

From the above, we can see that the master and infra nodes received `openshift-control-plane` profile and the worker node 'openshift-node' profile.
Let's change the label on the worker node, so that an ElasticSearch profile is applied.

```
user@b1: ~ $ oc get cm tuned-recommend -o yaml | grep elastic
    /var/lib/tuned/ocp-node-labels.cfg=.*node-role.kubernetes.io/elasticsearch=true
user@b1: ~ $ oc label node b3.lan node-role.kubernetes.io/elasticsearch=true
node "b3.lan" labeled
user@b1: ~ # oc logs tuned-l7q5b | grep applied
2018-08-14 10:45:36,483 INFO     tuned.daemon.daemon: static tuning from profile 'openshift-node' applied
2018-08-14 12:16:41,824 INFO     tuned.daemon.daemon: static tuning from profile 'openshift-node-es' applied
```

Let's remove the node label now.

```
user@b1: ~ $ oc label node b3.lan node-role.kubernetes.io/elasticsearch-
node "b3.lan" labeled
2018-08-14 10:45:36,483 INFO     tuned.daemon.daemon: static tuning from profile 'openshift-node' applied
2018-08-14 12:16:41,824 INFO     tuned.daemon.daemon: static tuning from profile 'openshift-node-es' applied
2018-08-14 12:21:11,149 INFO     tuned.daemon.daemon: static tuning from profile 'openshift-node' applied
```

Finally, let's change `openshift-control-plane` sysctl tuning by editing the parent openshift profile.

```
user@b1: ~ $ oc get cm tuned-profiles -o yaml | grep kernel.pid_max
      kernel.pid_max=>131072
user@b1: ~ $ sysctl kernel.pid_max
kernel.pid_max = 131072
user@b1: ~ $ oc edit cm tuned-profiles		# edit: kernel.pid_max=262144  (tuned 2.10 needed for =>)
user@b1: ~ $ sysctl kernel.pid_max
kernel.pid_max = 262144
```
