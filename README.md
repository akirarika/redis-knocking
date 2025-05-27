# redis-knocking

Protect your internal services - Redis knocking written in Go

We may need to deploy third-party services that either have their own authentication mechanisms or none at all. Exposing these services directly to the public internet is highly risky. Even if hackers can't guess passwords or find vulnerabilities, they could still crash your server via DDoS attacks.

Now, you might need a gatekeeper. It checks whether a client's IP address exists in a Redis Set to determine access. This allows you to create a unified verification page for these third-party services. Only visitors who pass your page's check (adding their IP to the whitelist) can access the services normally.

Here’s a simple breakdown of the scenario:

- There are multiple third-party services running on my server.

- - These services need to be accessible over the public internet, but only to me or authorized users.

These third-party services either lack authentication mechanisms or have their own user systems that cannot integrate with our internal systems.

Redis-knocking helps achieve the following:

- Users first access a designated login page. After successful authentication, their IP addresses are added to a Redis database (implementation handled by the developer).

- Redis-knocking acts as a gateway that checks the IP whitelist stored in Redis to decide whether to allow or deny access.

- Redis-knocking records the last access time of each IP to Redis, enabling automatic removal of inactive IPs from the whitelist (e.g., based on a predefined timeout configured by the developer).

## Installation in Container Images

Assuming your container runs in a `linux-amd64` environment:

```bash
RUN curl -o redis-knocking.tgz https://registry.npmjs.org/redis-knocking-linux-amd64/-/redis-knocking-linux-amd64-1.0.1.tgz \
    && mkdir -p __temp__redis-knocking \
    && tar zxvf  redis-knocking.tgz -C ./__temp__redis-knocking \
    && mv __temp__redis-knocking/package/bin/redis-knocking ./redis-knocking \
    && chmod +x ./redis-knocking \
    && rm -rf ./__temp__redis-knocking

CMD ./redis-knocking -script "npm run dev" -target "http://localhost:5173" -listen ":5174" -redis "redis://root:password@1.2.3.4:6379/0"
```

## Installation via npm

Using npm simplifies the installation. Choose the package matching your system architecture. For containers or servers, `redis-knocking-linux-amd64` is typically used:

```bash
npm i -g redis-knocking-darwin-amd64@1.0.1
npm i -g redis-knocking-darwin-arm64@1.0.1
npm i -g redis-knocking-freebsd-amd64@1.0.1
npm i -g redis-knocking-freebsd-arm64@1.0.1
npm i -g redis-knocking-linux-386@1.0.1
npm i -g redis-knocking-linux-amd64@1.0.1
npm i -g redis-knocking-linux-arm@1.0.1
npm i -g redis-knocking-linux-arm64@1.0.1
npm i -g redis-knocking-windows-386@1.0.1
npm i -g redis-knocking-windows-amd64@1.0.1
npm i -g redis-knocking-windows-arm64@1.0.1
```

After installation, run it (using `redis-knocking-linux-amd64` as an example):

```bash
redis-knocking-linux-amd64 -script "npm run dev" -target "http://localhost:5173" -listen ":5174" -redis "redis://root:password@1.2.3.4:6379/0"
```

## Redis Key Configuration

By default, the service checks the Redis Set `ip-allowed` to determine authorized IPs. To customize, use the `-set` parameter:

> Note: Internal IPs are automatically allowed and do not require whitelisting.

```bash
-set "ip-allowed-custom"
```

## History Tracking

By default, the service records the last access time of allowed IPs in a Redis hash under the key `ip-history`. This can be used to implement custom logic (e.g., auto-removing inactive IPs). Customize with the `-history` parameter:

> Note: Internal IPs are not tracked here as they are always allowed.

```bash
-history "ip-history-custom"
```

## IP Detection Method

If the service runs behind a gateway/proxy, configure the HTTP header field to extract the real client IP:

```bash
-ipHeader "X-Forwarded-For"
```

## Redirect Instead of Blocking

By default, unauthorized connections are closed. To redirect users to an authentication page instead, use the `-redirect` parameter:

```bash
-redirect "https://www.google.com"
```

## Verbose Logging

By default, the service does not log detailed IP access information. Enable verbose logging with the `-detail` parameter:

```bash
-detail "enabled"
```
