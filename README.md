# redis-knocking

## 在容器镜像中安装

我们假设你的容器是运行在 `linux-amd64` 环境中：

```
curl -o redis-knocking.tgz https://registry.npmjs.org/redis-knocking-linux-amd64/-/redis-knocking-linux-amd64-1.0.0.tgz \
    && mkdir -p __temp__redis-knocking \
    && tar zxvf  redis-knocking.tgz -C ./__temp__redis-knocking \
    && mv __temp__redis-knocking/package/bin/redis-knocking ./redis-knocking \
    && rm -rf ./__temp__redis-knocking

CMD ./redis-knocking -target "http://localhost:5173" -listen ":5174" -redis "redis://root:password@1.2.3.4:6379/0" -script "npm run dev"
```

## 使用 npm 安装

使用 npm 可以简化我们的安装过程，选择你的系统对应的架构，对于容器或服务器部署来说，通常是选择 `redis-knocking-linux-amd64` 即可。

```
npm i -g redis-knocking-darwin-amd64@1.0.0
npm i -g redis-knocking-darwin-arm64@1.0.0
npm i -g redis-knocking-freebsd-amd64@1.0.0
npm i -g redis-knocking-freebsd-arm64@1.0.0
npm i -g redis-knocking-linux-386@1.0.0
npm i -g redis-knocking-linux-amd64@1.0.0
npm i -g redis-knocking-linux-arm@1.0.0
npm i -g redis-knocking-linux-arm64@1.0.0
npm i -g redis-knocking-windows-386@1.0.0
npm i -g redis-knocking-windows-amd64@1.0.0
npm i -g redis-knocking-windows-arm64@1.0.0
```

完成后，运行它，下面以 `redis-knocking-linux-amd64` 举例：

```
redis-knocking-linux-amd64 -target "http://localhost:5173" -listen ":5174" -redis "redis://root:password@1.2.3.4:6379/0" -script "npm run dev"
```

## IP 获取方式

我们的程序可能运行在其他网关背后，因此获取不到用户的真实 IP。我们可以配置读取哪个 HTTP 请求头中的字段来记录 IP：

```
-ipHeader "X-Forwarded-For"
```

## Redis Key

默认情况下，程序会读取 `ip-allowed` 这个 Redis Key 来判断用户是否授权。如果需要自定义，可以添加 `-set` 参数：

```
-set "ip-allowed-custom"
```

## 重定向而非断开连接

默认情况下，会直接端口链接。我们可以使用户没有授权时，调整到某个引导用户授权的页面。追加 `-redirect` 参数即可：

```
-redirect "https://www.google.com"
```

## 显示详细日志

默认情况下，程序不输出访问的 IP 等信息。如果想要显示这些信息以便于调试，可以添加 `-detail` 参数：

```
-detail "enabled"
```
