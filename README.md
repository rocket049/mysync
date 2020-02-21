# mysync - 单向同步软件

#### 关于 `v3.1.1`
以很小的代价消除了上一版本的副作用。同时改进了客户端，减少了计算量。

#### 关于 `v3.1.0`
该版本和 `v3.0.1` 兼容，改进之处：服务端使用文件 `_desc.json` 保存各个文件的 `MD5`。
- 优点是：免去了每次同步时的重复计算，当目录中有大文件时，明显减轻了服务器负担。
- 副作用：如果手动改动了服务端的文件，必须删除 `_desc.json`，否则服务器无法感知文件的改变。

#### 项目介绍
这是基于RSA、AES256加密验证的单向同步软件，从客户端同步到服务端，自动跳过第一级目录中以`.`或`_`开头的文件和目录。

服务器和客户端的通讯使用`TLS`安全连接进行`RPC`调用。

#### 源代码地址
- [github.com](https://github.com/rocket049/mysync)
- [gitee.com](https://gitee.com/rocket049/mysync)

#### 百度网盘下载
[https://pan.baidu.com/s/103cgeSFOmPZFvVZOQYdDPw](https://pan.baidu.com/s/103cgeSFOmPZFvVZOQYdDPw)

#### 算法说明
- 首先由客户端获取本地目录的文件列表，并且逐一计算各个文件的`MD5`值，然后把文件名、`MD5`列表发送到服务器。
- 接着服务器也计算服务器上的文件、`MD5`列表，根据各个文件的`MD5`与客户端上传的列表进行比较，找出被客户端修改、新增、删除了的文件，然后删除已被客户端删除的文件，并且向客户端返回修改、新增文件列表。
- 最后客户端把已经修改、新增的文件压缩打包后上传到服务器，服务器解开压缩包，更新服务器上的文件夹内容。

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



