# mysync

#### 项目介绍
基于RSA、AES256加密验证的单向同步软件，从客户端同步到服务端，自动跳过第一级目录中以`.`或`_`开头的文件和目录。

#### 软件架构
软件架构说明


#### 安装教程

1. xxxx
2. xxxx
3. xxxx

#### 使用说明
linux
1. mysyncd : 服务器，使用配置文件目录：`HOME/mysyncd/`，`mykey.pub`是客户端RSA公钥,`mykey.json`指明同步目录，
`mykey`名字可以改变，和客户端的配置有关。
2. mysync : 客户端，使用配置文件目录：`HOME/mysync/`，`mykey.key`是客户端RSA私钥，`local.json`指明本地目录`root`、
服务器地址`host`，标识符`key`，标识符必须和服务器上的`.pub`、`.json`文件前面的名字相同，和本地的私钥文件`.key`文件的
名字也必须相同。
3. genkey：密码工具，生成一对RSA密钥`name.pub`、`name.key`，用法：`genkey -k name`。

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