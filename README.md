# offlinepush
一个分布式的离线推送系统

文档：https://www.yuque.com/u37556924/ttv9rp/ewgh5zlt1wmuf9fi?singleDoc# 《OfflinePush》
## 项目部署
### 本地部署
* 打开终端1,输入docker ps确认Docker已经正常启动

*  在终端1输入make deploy_3rd启动第三依赖

* 打开终端2,等1~3秒输入make dev启动项目
### 集群模式
##### 初次部署： 
* 打开终端1,输入docker ps确认Docker已经正常启动
* 在终端1输入make deploy_3rd启动第三依赖
* 打开终端2,等1~3秒输入make init_k8s启动项目
##### 更新部署：
* 打开终端1,输入docker ps确认Docker已经正常启动
* 在终端1输入make deploy_3rd启动第三依赖
* 打开终端2,等1~3秒输入make update_k8s启动项目