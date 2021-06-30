# oracle_free
oracle免费服务器VM.Standard.A1.Flex配置升级,最高可升级到4ocpu,24g内存

# 使用

## 新建配置文件
```
mkdir ~/.oci
touch ~/.oci/config
```

## 添加API
* 登录oracle后台管理，右上角 用户设置，左下方api密钥，添加api密钥
* 点击下载私钥密钥
* 点击添加
* 将页面上的信息复制到~/.oci/config
* 修改~/.oci/config中key_file的路径（刚刚下载的私钥存放路径）

## 运行
### 方式一
```
go run main.go
```

### 方式二
下载可执行文件，直接运行(暂未打包)