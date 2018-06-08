# mysync

#### 项目介绍
基于RSA、AES256加密验证的单向同步软件，从客户端同步到服务端，自动跳过第一级目录中以`.`或`_`开头的文件和目录。

#### 软件架构
软件架构说明：

服务器和客户端的通讯使用`RPC`和`HTTP`，`RPC`进行控制，`HTTP`上传文件。

#### 安装教程

1. 把`src`目录下的几个目录复制到`GOPATH/src`下面，然后编译。
2. “附件”中有编译好的win32版本和linux_amd64版本，静态链接的，解压后可以直接运行。
3. xxxx

#### 使用说明

*linux*

`linux`目录中有配置文件、Makefile样本。

1. mysyncd : 服务器，使用配置文件目录：`HOME/mysyncd/`，`mykey.pub`是客户端RSA公钥,`mykey.json`指明同步目录，
`mykey`名字可以改变，和客户端的配置有关。默认绑定地址为`":6080"`,可使用参数`host`改变，参数格式：`-host IP:PORT`。
2. mysync : 客户端，使用配置文件目录：`HOME/mysync/`，`mykey.key`是客户端RSA私钥，默认配置文件`local.json`指明本地目录`root`、
服务器地址`host`，标识符`key`，标识符必须和服务器上的`.pub`、`.json`文件前面的名字相同，和本地的私钥文件`.key`文件的
名字也必须相同。配置文件可以指定：`-conf name`代表使用名字为`name.json`的配置文件。
3. genkey：密码工具，生成一对RSA2048密钥`name.pub`、`name.key`，用法：`genkey -k name`。

*windows*

`windows`目录中有配置文件、Makefile（在`linux`下交叉编译）样本。

和`linux`的区别：配置文件放在可执行文件`mysyncd.exe`、`mysync.exe`同一目录下的`conf`目录中。

#### 参与贡献

1. Fork 本项目
2. 新建 Feat_xxx 分支
3. 提交代码
4. 新建 Pull Request


#### 码云特技

1. 使用 Readme\_XXX.md 来支持不同的语言，例如 Readme\_en.md, Readme\_zh.md
2. 码云官方博客 [blog.gitee.com](https://blog.gitee.com)
3. 你可以 [https://gitee.com/explore](https://gitee.com/explore) 这个地址来了解码云上的优秀开源项目
4. [GVP](https://gitee.com/gvp) 全称是码云最有价值开源项目，是码云综合评定出的优秀开源项目
5. 码云官方提供的使用手册 [http://git.mydoc.io/](http://git.mydoc.io/)
6. 码云封面人物是一档用来展示码云会员风采的栏目 [https://gitee.com/gitee-stars/](https://gitee.com/gitee-stars/)