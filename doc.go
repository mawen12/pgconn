// Package pgconn is a low-level PostgreSQL database driver.
/*
pgconn provides lower level access to a PostgreSQL connection than a database/sql or pgx connection. It operates at
nearly the same level is the C library libpq.

Establishing a Connection

Use Connect to establish a connection. It accepts a connection string in URL or DSN and will read the environment for
libpq style environment variables.

Executing a Query

ExecParams and ExecPrepared execute a single query. They return readers that iterate over each row. The Read method
reads all rows into memory.

Executing Multiple Queries in a Single Round Trip

Exec and ExecBatch can execute multiple queries in a single round trip. They return readers that iterate over each query
result. The ReadAll method reads all query results into memory.

Context Support

All potentially blocking operations take a context.Context. If a context is canceled while the method is in progress the
method immediately returns. In most circumstances, this will close the underlying connection.

The CancelRequest method may be used to request the PostgreSQL server cancel an in-progress query without forcing the
client to abort.
*/

/*
pgconn 包是一个低级的 PostgreSQL 数据库驱动程序。

pgconn 提供了比 database/sql 或 pgx 连接更低级别访问 PostgreSQL 连接的功能。
它的操作级别几乎与 C 库 libpq 相同。

# 建立连接

使用 Connect 来建立连接。它接受 URL 或 DSN 格式的连接字符串，并将读取环境中的 libpq 样式环境变量。

# 执行查询

ExecParams 和 ExecPrepared 执行单个查询。他们返回迭代每行的读取器。Read 方法将所有行读入内存。

# 在单个往返中执行多个查询

Exec 和 ExecBatch 可以在单个往返中执行多个查询。他们返回迭代每个查询结果的读取器。
ReadAll 方法将所有查询结果读入内存。

# 上下文支持

所有可能阻塞的操作都接受 context.Context。如果在方法执行过程中取消了上下文，方法将立即返回。
在大多数情况下，这将关闭底层连接。

CancelRequest 方法可用于请求 PostgreSQL 服务器取消正在进行的查询，而不强制客户端中止。

# 处理流程

	connect
		通过 receiveMessage 循环读取消息，直到读取到 ReadyForQuery 或者 ErrorResponse。

		处理以下消息：
			BackendKeyData

			AuthenticationOk
				无需任何处理

			AuthenticationCleartextPassword
				客户端需要传递明文密码

			AuthenticationMD5Password
				客户端需要传递MD5加密密码

			AuthenticationSASL

			AuthenticationGSS

			ReadyForQuery
				status = Idle，可以接受请求执行后续处理。

			ParameterStatus
				不做处理，由 receiveMessage 做处理

			NoticeResponse
				不做处理，由 receiveMessage 做处理

			ErrorResponse
				关闭连接，status = Closed

			default
				关闭连接，status = Closed

	receiveMessage
		通过 peekMessage 来读取消息，并在读取之后，清除 peekedMsg 消息，
		确保下次读取的是新的消息。

		处理以下消息：
			ReadyForQuery
				保存后端的事务状态标识

			ParameterStatus
				保存运行时参数信息

			ErrorResponse
				如果是 FATAL 级别，则关闭连接，并返回错误。

			NoticeResponse
				触发 OnNotice 回调

			NotifcationResponse
				触发 OnNotification 回调

	peekMessage
		通过 Frontend.Receive 读取消息，然后将读取到的消息保存到 peekedMsg 中。
		该方法会缓存最近一次读取的消息。对于重复调用会返回最近读取的消息。
*/
package pgconn
