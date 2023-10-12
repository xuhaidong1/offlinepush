#!/bin/bash

# 版本号文件,用于构建镜像
VERSION_FILE=$1

# 如果版本文件不存在，创建一个初始版本号
if [ ! -f "$VERSION_FILE" ]; then
  echo "0.0.0" > "$VERSION_FILE"
fi

# 读取当前版本号
current_version=$(cat "$VERSION_FILE")

# 解析版本号的三个部分
major=$(echo "$current_version" | cut -d. -f1)
minor=$(echo "$current_version" | cut -d. -f2)
patch=$(echo "$current_version" | cut -d. -f3)

# 自增版本号的 patch 部分
patch=$((patch + 1))

# 如果 patch 部分超过 999，将其重置为 0 并自增 minor 部分
if [ "$patch" -gt 99 ]; then
  patch=0
  minor=$((minor + 1))
fi

# 如果 minor 部分超过 999，将其重置为 0 并自增 major 部分
if [ "$minor" -gt 99 ]; then
  minor=0
  major=$((major + 1))
fi

# 构建新的版本号
new_version="$major.$minor.$patch"

# 将新版本号写入文件
echo "$new_version" > "$VERSION_FILE"
