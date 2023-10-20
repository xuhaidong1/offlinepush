#!/bin/bash

cp $1/deploy/k8s/k8s-offlinepush-template.yaml $1/deploy/k8s/k8s-offlinepush.yaml
# macos 特殊sed命令
sed -i '' -e "s/IMAGE_TAG/v`cat $1/deploy/version/version.txt`/g" $1/deploy/k8s/k8s-offlinepush.yaml