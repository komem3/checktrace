# CheckTrace

grpc-gateway で GCP の TraceID がリクエストから伝搬されることを確認する。

- `httpserver` シンプルな http server
- `grpcinternal` 同一インスタンスで grpc gateway と grpc server があるとき
- `grpcserver` + `grpcgateway` server と gateway が別インスタンスの時に trace を伝搬させる
