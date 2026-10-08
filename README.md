# XJTLU Srun Client

Upstream: [SZU Srun Client](https://github.com/Caterpie771881/szu_srun_client)

一个用于在命令行环境下登陆XJTLU校园网 (❓ 有线(未测试)/✅ WiFi) 的客户端, 适用于 srun 认证系统 (drcom 认证系统请移步 [(过于简单，暂无计划)](https://github.com/wujinjun-MC))

~~需要 python3 环境, 由于认证逻辑比较复杂, 所以没用 shell 脚本来写~~

最新客户端使用 golang 编写, 无需依赖 python 环境即可实现跨平台

若所在平台支持 python3, 也可以使用 `PyClient` 文件夹下的脚本

## 使用

### golang 版客户端

#### 快速启动 (hidden)
> 😞 未登录时IPv4/IPv6均处于隔离状态 (出站/入站失败)，无法实现上游"快速启动"功能

#### 手动下载
TODO: Add CI for GoClient

#### 手动构建

```bash
cd GoClient
go build
```


### python 版客户端

保证所在机器有 python3 环境, 将本项目中的 [PyClient](./PyClient/) 文件夹复制到机器上

然后执行以下命令

```
cd PyClient & chmod +x main.py & python3 main.py
```

## 配置 & 命令行参数

同时支持 YAML 配置文件 `config.yml` 和命令行参数(优先)。

示例配置文件:

```yaml
# Mode: 1 表示上线， 2 表示下线
mode: "1"
# Username
username: "your username (without @xjtlu.edu.cn or @student.xjtlu.edu.cn)"
# Password | TODO: encrypt
password: "your password"
# Operating System, Windows|Linux|Android|iOS ...
#os: "Windows"
# IP: 内网IP (获取: ifconfig/ip addr show)
#ip: "10.13.128.?"
# Interface: Linux 网卡名称，仅 Linux 上的 Python 客户端支持
#interface: "eth0"
# AC ID: WiFi = 0, 有线 = ?
#ac_id: "0"
# Encrypt version: SRun internal value
#enc_ver: "srun_bx1"
# 【新增】 指定接口 (仅支持Linux, Python客户端)
#interface: wlan0
```

mode,username,password 必须填写，不填写时启动客户端会自动询问。

Use `--mode`, `--username`, `--password`, `--os`, `--ip`, `--interface`, `--ac_id`, and `--enc_ver` with the Python client.
`interface` binds all client

HTTP requests to the named Linux network interface through `SO_BINDTODEVICE`.

Specifying `interface` on a non-Linux system or with the Go client exits with an error.

The Go client uses the same flags with a single leading dash and uses `EncVer` as the YAML key (it also accepts `enc_ver`).

Use `--config` or `-config` to select another YAML file.

When `ip` is omitted, it is obtained automatically.

## Goals

- [ ] SystemD service: Auto test connectivity and login (After that you may disable "MAC auth" to avoid MAC manipulation attack)
- [ ] NetworkManager dispatcher: Auto connect when specified interface(s) are up