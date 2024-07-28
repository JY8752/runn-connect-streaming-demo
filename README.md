# runn-connect-streaming-demo

ConnectのServer StreamingとRedisのPubSubで構成したリアルタイム処理をrunnでテストするためのデモです。

## Connectサーバーの起動

```
go run main.go
```

## Redisの起動

```
docker run --name runn-demo-redis -d -p 6379:6379 redis:7.2.5
```

## ランブックの実行

```
make books
```

## リンク

[Protobufモジュール](https://buf.build/jyapp/runndemo)

[zenn記事](https://zenn.dev/jy8752/articles/685f7001e3a351)
