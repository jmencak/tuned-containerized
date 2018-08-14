#!/bin/sh

oc delete project tuned
oc delete scc tuned
oc delete clusterrolebinding cluster-reader-tuned
oc project default
