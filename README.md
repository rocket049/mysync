# mysync - 单向同步软件

#### 项目介绍
这是基于RSA、AES256加密验证的单向同步软件，从客户端同步到服务端，自动跳过第一级目录中以`.`或`_`开头的文件和目录。

服务器和客户端的通讯使用`TLS`安全连接进行`RPC`调用。

#### 百度网盘下载
[https://pan.baidu.com/s/103cgeSFOmPZFvVZOQYdDPw](https://pan.baidu.com/s/103cgeSFOmPZFvVZOQYdDPw)

#### 系统构架
本软件有4个程序构成：

1. `mysyncd` - 服务端程序，可以使用参数改变端口，查询参数：`mysyncd -h`。
2. `mysync` - 客户端程序。
3. `genca` - 生成自签名的`TLS`证书对`servername-cert.pem、servername-key.pem`，用法参考：`genca -h`。
4. `genkey` - 生成一对RSA2048密钥`name.pub`、`name.key`，用法：`genkey -k name`。

#### 配置文件路径
- 服务器：在`Linux`系统上是：`HOME/config/mysyncd/`；在`Windows`系统上是：`/path/to/mysyncd/config/mysyncd/`
- 客户端：在需要同步的目录中的子目录 `_mysync`

#### 配置`TLS`证书：

1. 用`genca`程序生成`servername-cert.pem、servername-key.pem`。
2. 服务端：把`servername-cert.pem、servername-key.pem`复制到`config/mysyncd/rootcas`，改名为`cert.pem、key.pem`；
3. 客户端：把`servername-cert.pem`复制到`_mysync`中，改名为`cert.pem`。

#### 配置服务器`mysyncd`
把客户端`RSA`公钥`mykey.pub`复制到`config/mysyncd`目录中,编辑对应的`mykey.json`文件指明同步目录，
注意`JSON`文件和`.pub`文件的名字是一一对应的。`mykey`名字可以改变，和客户端的配置相对应。默认绑定地址为`":6080"`,可使用`mysyncd`程序的参数`host`改变，参数格式：`-host IP:PORT`。

服务端可以配置多对`mykey.pub、mykey.json`

#### 配置客户端`mysync`
把客户端的RSA私钥`mykey.key`复制到`_mysync`目录中，编辑配置文件`config.json`指明
服务器地址`host`，标识符`key`，标识符必须和服务器上的`.pub`、`.json`文件前面的名字以及本地的私钥文件`.key`文件的名字相同。

#### 服务端配置目录结构
```
config/mysyncd/
├── mykey.json
├── mykey.pub
└── rootcas
    ├── cert.pem
    └── key.pem

```

#### 客户端配置文件结构
```
_mysync/
├── cert.pem
├── config.json
└── mykey.key

```



