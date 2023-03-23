# go-chatgpt-bot
ChatGPT，对接个人微信

## 使用方法
### 编译可执行文件
```shell
make deps # 安装倚赖
make bin # 编译
```

### 运行方法
#### 1. 配置文件
在https://platform.openai.com/account/api-keys 下获取自己的SECRET KEY，并填入config.yaml文件
```shell
cp etc/config.yaml.example config.yaml
```
#### 2. 运行
```shell
./bin/go-chatgpt-bot start -c config.yaml

或者 
bash scripts/start.sh
## 运行以后会在终端显示二维码链接，打开并用微信扫描登录即可
访问下面网址扫描二维码登录
https://login.weixin.qq.com/qrcode/IcqL-5PuXw==
```
