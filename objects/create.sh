#!/bin/sh

for f in *.yaml ; do
  oc create -f $f
done
