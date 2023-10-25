APP_PATH:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
SCRIPTS_PATH:=$(APP_PATH)/scripts

.PHONY:	setup
setup:
	@echo "初始化开发环境......"
	@find "$(SCRIPTS_PATH)" -type f -name '*.sh' -exec chmod +x {} \;
	@bash $(SCRIPTS_PATH)/setup/setup.sh
	@$(MAKE) tidy

# 依赖清理
.PHONY: tidy
tidy:
	@go mod tidy

# 代码风格
.PHONY: fmt
fmt:
	@goimports -l -w $$(find . -type f -name '*.go' -not -path "./.idea/*" -not -path "./cmd/ioc/*")
	@gofumpt -l -w $$(find . -type f -name '*.go' -not -path "./.idea/*" -not -path "./cmd/ioc/*")

# 单元测试
.PHONY:	ut
ut:
	@go test -race -cover -coverprofile=unit.out -failfast -shuffle=on ./...

# 集成测试
.PHONY: it
it:
#	@make dev_3rd_down
#	@make dev_3rd_up
#	@go test -tags=integration -race -cover -coverprofile=integration.out -failfast -shuffle=on ./test/...
#	@make dev_3rd_down

# 端到端测试
.PHONY: e2e
e2e:
#	@make dev_3rd_down
#	@make dev_3rd_up
#	@go test -tags=e2e -race -cover -coverprofile=e2e.out -failfast -shuffle=on ./test/...
#	@make dev_3rd_down

# 启动本地研发 docker 依赖
.PHONY: dev_3rd_up
dev_3rd_up:
	@docker compose -f ./scripts/deploy/depend/offlinepush/docker-compose.yaml up -d

.PHONY: dev_3rd_down
dev_3rd_down:
	@docker compose -f ./scripts/deploy/depend/offlinepush/docker-compose.yaml down -v

.PHONY: dev
dev:
	@echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> 本地部署单体 <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
	#@make dev_3rd_down
	#@make dev_3rd_up
	@go run ./cmd/main --config=conf/dev.toml


PREFIX := "xuhaidong/offlinepush:v"
IMAGE_NAME := "xuhaidong/offlinepush"
IMAGE_ID := $(shell docker images |grep $(IMAGE_NAME)|head -n 1|awk '{print $$3}')
CONTAINER_ID := $(shell docker ps |grep "/app/offlinepush"|awk '{print $$1}')
OLD_IMAGE := $(PREFIX)$(shell cat $(SCRIPTS_PATH)/deploy/version/version.txt)

#-------k8s更新部署点这个-------------------------------------------------------
.PHONY: update_k8s
update_k8s:
	#这两个要分开写，要不然第2步获取到的是旧版本号
	@make incr_version
	@make deploy_image_update
	@docker rmi $(OLD_IMAGE)||true
#-------k8s更新部署点这个--------------------------------------------------------

.PHONY: incr_version
incr_version:
	# 增加版本号
	@bash $(SCRIPTS_PATH)/deploy/version/increase_version.sh $(SCRIPTS_PATH)/deploy/version/version.txt


.PHONY: deploy_image_update
deploy_image_update:
	@echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> 本地部署集群 <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
	# 把上次编译的东西删掉
	@rm ./compile_out/offlinepush || true
	# 运行一下 go mod tidy，防止 go.sum 文件不对，编译失败
	@make tidy
	# 指定编译成在 ARM 架构的 linux 操作系统上运行的可执行文件，
	# -tags=k8s 暂时不需要
	@GOOS=linux GOARCH=arm go build -tags=k8s -o ./compile_out/offlinepush ./cmd/main

	# 把新版本镜像tag写入yaml
	@bash $(SCRIPTS_PATH)/deploy/version/fill_k8s_yaml.sh $(SCRIPTS_PATH)
	# 构建新版本镜像
	@docker build -t $(PREFIX)$(shell cat $(SCRIPTS_PATH)/deploy/version/version.txt) -f $(SCRIPTS_PATH)/deploy/k8s/Dockerfile .
	# 滚动更新
	@kubectl set image deployment/offlinepush  offlinepush=$(PREFIX)$(shell cat $(SCRIPTS_PATH)/deploy/version/version.txt)


.PHONY: deploy_3rd
deploy_3rd:
	@kubectl apply -f $(SCRIPTS_PATH)/deploy/k8s/k8s-redis.yaml
	@kubectl apply -f  $(SCRIPTS_PATH)/deploy/k8s/k8s-etcd.yaml

.PHONY: deploy_k8s
deploy_k8s:
	@kubectl apply -f  $(SCRIPTS_PATH)/deploy/k8s/k8s-offlinepush.yaml

.PHONY: init_k8s
ini_k8s:
	@make incr_version
	@make deploy_k8s
	@echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> 本地部署集群 <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
	# 把上次编译的东西删掉
	@rm ./compile_out/offlinepush || true
	# 运行一下 go mod tidy，防止 go.sum 文件不对，编译失败
	@make tidy
	# 指定编译成在 ARM 架构的 linux 操作系统上运行的可执行文件，
	# -tags=k8s 暂时不需要
	@GOOS=linux GOARCH=arm go build -tags=k8s -o ./compile_out/offlinepush ./cmd/main

	# 把新版本镜像tag写入yaml
	@bash $(SCRIPTS_PATH)/deploy/version/fill_k8s_yaml.sh $(SCRIPTS_PATH)
	# 构建新版本镜像
	@docker build -t $(PREFIX)$(shell cat $(SCRIPTS_PATH)/deploy/version/version.txt) -f $(SCRIPTS_PATH)/deploy/k8s/Dockerfile .



replicas:
	#先拿到 正在部署的版本
	@kubectl describe deploy offlinepush|grep Image
	#修改k8s-offlinepush.yaml 的replicas 镜像版本填入正在部署的版本 正常都不需要修改
	@kubectl apply -f  $(SCRIPTS_PATH)/deploy/k8s/k8s-offlinepush.yaml