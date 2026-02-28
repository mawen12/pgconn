# Conn Design

分析 `pgconn` 连接的逻辑。

## connect

1. 解析连接字符串
2. 与服务器建立连接
3. 进行认证

## 架构

PgConn#Exec
 -> ResultReader 
 -> PgConn 
 -> 