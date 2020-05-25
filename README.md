# eventdrip-media-server

Publish RTMP to `rtmp://localhost:1935/stream/<PRIVATE_STREAM_KEY>`
Access HLS stream at `http://localhost:7935/stream/<PUBLIC_STREAM_ID>.m3u8`

## Run in Docker
```
docker build --tag eventdrip:1.0 .
docker run --network="host" -e AUTH_HOST="http://localhost:8001/auth" eventdrip:1.0
```
```
node ./web-api/app.js
```

## Web API

### `POST /auth`
Validate a stream key and return the corresponding manifest ID.
```javascript
{
   StreamKey: "STREAM_KEY",
}
```
```javascript
{
   ManifestID: "MANIFEST_ID"
}
```

### `GET /new`
Create a new stream key / manifest ID pair.
```javascript
{
   streamKey: "STREAM_KEY",
   manifestID: "MANIFEST_ID"
}
```

## Deploy Swarm
```
docker stack deploy -c eventdrip-stack.yml eventdrip
```
