# eventdrip-media-server


## Install
Install system dependencies:
```
sudo apt update
sudo apt install -y autoconf gnutls-dev
```

Install libavcodec (FFMPEG), NASM, VideoLAN:
```
bash ./scripts/install_ffmpeg.sh
export PKG_CONFIG_PATH=$HOME/compiled/lib/pkgconfig:$PKG_CONFIG_PATH
```

## Run
```
go run cmd/lpms/main.go
```

Publish to `rtmp://localhost:1935/stream/test`
Access stream at `http://localhost:7935/stream/test_hls.m3u8`

## Run in Docker
```
sudo docker build --tag eventdrop:1.0 .
sudo docker run -p 1935:1935 -p 7935:7935 -p 8001:8001 eventdrop:1.0
```

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
