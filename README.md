# eventdrip-media-server

Publish RTMP to `rtmp://localhost:1935/stream/<PRIVATE_STREAM_KEY>`
Access HLS stream at `http://localhost:7935/stream/<PUBLIC_STREAM_ID>.m3u8`

## Run in Docker
```
docker build --tag eventdrip:1.0 .
docker run -p 1935:1935 -p 7935:7935 eventdrip:1.0
```

## Web API

Validate a stream key and return the corresponding manifest ID.
### `POST /auth`
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

Create a new stream key / manifest ID pair.
## `GET /new`
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
