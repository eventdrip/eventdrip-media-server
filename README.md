# eventdrip-media-server


## Install
Install system dependencies:
```
sudo apt update
sudo apt install -y autoconf gnutls-dev
```

Install libavcodec (FFMPEG), NASM, VideoLAN:
```
bash ./scripts/install.sh
```

## Run
```
go run cmd/lpms/main.go
```

Publish to `rtmp://localhost:1935/stream/test`
Access stream at `http://localhost:7935/stream/test_hls.m3u8`

## Web API
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

