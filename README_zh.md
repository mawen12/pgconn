---

该版本被 pgx `v4` 使用。在 pgx `v5` 中，它是 https://github.com/jackc/pgx 仓库的一部分。

---

# pgconn

pgconn 包是一个低级 PostgreSQL 数据库驱动。它与 C 库 libpq 的级别几乎相同。
它的主要目的是作为更高级库，如 https://github.com/jackc/pgx。
应用程序应该使用高高级库来处理常规查询，只有在需要对 PostgreSQL 功能进行底层访问时
才直接使用 pgconn。

## 示例用法

```go
pgConn, err := pgconn.Connect(context.Background(), os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatalln("pgconn failed to connect:", err)
}
defer pgConn.Close(context.Background())

result := pgConn.ExecParams(context.Background(), "SELECT email FROM users WHEREE id = $1", [][]byte{[]byte("123")}, nil, nil, nil)
for result.NextRow() {
    fmt.Println("User 123 has email:", string(result.Values()[0]))
}
_, err := result.Close()
if err != nil {
    log.Fatalln("failed reading result:", err)
}
```

## 测试

pgconn 的测试i徐阿哟一个 PostgreSQL 数据库。它将连接到 `PGX_TEST_CONN_STRING` 环境变量中
指定的数据库。`PGX_TEST_CONN_STRING` 环境变量可以是 URL 或 DSN。此外，标准的 `PG*` 环境变量
也被尊重。考虑使用 [direnv](https://github.com/direnv/direnv) 来简化环境变量处理。

### 示例测试环境

连接你的 PostgreSQL 服务器并运行：

```
create database pgx_test;
```

现在你可以运行测试：

```bash
PGX_TEST_CONN_STRING="host=/var/run/postgresql dbname=pgx_test" go test ./...
```

### 连接和认证测试

Pgconn 支持多种连接类型和认证方式。这些测试是可选的。它们只有在设置了适当的环境变量时才会运行。
运行 `go test -v | grep SKIP` 来查看是否有任何测试被跳过。
大部分开发者无需运行这些测试。它们主要用于确保 pgconn 在各种环境中正确连接和认证。