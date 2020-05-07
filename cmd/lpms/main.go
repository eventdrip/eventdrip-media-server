package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/livepeer/lpms/segmenter"

	"github.com/google/uuid"
	"github.com/livepeer/lpms/core"
	"github.com/livepeer/lpms/stream"
	"github.com/livepeer/m3u8"
)

const authHost = "http://localhost:8001/auth"
const hlsManifestLength uint = 3
const hlsWaitTime = time.Second * 10

type JSONAuthRequest struct {
	StreamKey string
}
type JSONAuthResponse struct {
	ManifestID string
}

type CustomAppData struct {
	streamID         string
	manifestID       string
	variantID        string
	cancelSegmenting context.CancelFunc
}

func (s *CustomAppData) StreamID() string {
	return s.streamID
}

type RTMPStreamData struct {
	rtmpStream stream.RTMPVideoStream
}

type HLSMasterManifest struct {
	manifest *stream.BasicHLSVideoManifest
}
type HLSVariantData struct {
	hlsStream stream.HLSVideoStream
}

type RTMPStreamMap map[string]RTMPStreamData
type HLSMasterMap map[string]HLSMasterManifest
type HLSVariantMap map[string]HLSVariantData

func generateUUID() string {
	id, err := uuid.NewRandom() // v4 secure UUID
	if err != nil {
		panic(err)
	}
	return id.String()
}

func parseRTMPStreamKey(path string) (string, bool) {
	var streamID string
	regex, _ := regexp.Compile("\\/stream\\/([[:alpha:]]|\\d|\\-)*")
	match := regex.FindString(path)
	if match != "" {
		streamID = strings.Replace(match, "/stream/", "", -1)
		return streamID, true
	}
	return "", false
}

func parseHLSManifestID(path string) (string, bool) {
	regex, _ := regexp.Compile("\\/stream\\/((?:[[:alpha:]]|\\d|\\-)*)_?.*(?:(?:\\.m3u8)|(?:\\.ts))")
	matches := regex.FindStringSubmatch(path)
	if len(matches) > 1 {
		return matches[1], true
	}
	return "", false
}

func parseHLSSegmentName(path string) (string, bool) {
	var segName string
	regex, _ := regexp.Compile("\\/stream\\/.*\\.ts")
	match := regex.FindString(path)
	if match != "" {
		segName = strings.Replace(match, "/stream/", "", -1)
		return segName, true
	}
	return "", false
}

func main() {
	opts := core.LPMSOpts{
		RtmpAddr: "0.0.0.0:1935",
		HttpAddr: "0.0.0.0:7935",
		WorkDir:  "/tmp",
	}
	lpms := core.New(&opts)
	rtmpStreams := make(RTMPStreamMap)
	hlsMasterManifests := make(HLSMasterMap)
	hlsVariants := make(HLSVariantMap)

	// handle RTMP publish
	lpms.HandleRTMPPublish(
		createRTMPStreamIDHandler(),
		createRTMPStreamHandler(lpms, rtmpStreams, hlsMasterManifests, hlsVariants),
		createRTMPStreamEndHandler(rtmpStreams, hlsMasterManifests, hlsVariants))

	lpms.HandleRTMPPlay(
		createRTMPPlayHandler(rtmpStreams))

	// handle HLS play
	lpms.HandleHLSPlay(
		createHLSMasterPlaylistHandler(hlsMasterManifests),
		createHLSMediaPlaylistHandler(hlsVariants),
		createHLSSegmentHandler(hlsVariants))
	lpms.Start(context.Background())
	glog.Info("Server running...")
}

func authenticateRTMPPublish(streamKey string) (string, error) {
	b, err := json.Marshal(JSONAuthRequest{streamKey})
	if err != nil {
		return "", err
	}
	resp, err := http.Post(authHost, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var m JSONAuthResponse
	err = json.Unmarshal(body, &m)
	if err != nil {
		return "", err
	}

	return m.ManifestID, nil
}

func createRTMPStreamIDHandler() func(*url.URL) stream.AppData {
	return func(url *url.URL) stream.AppData {
		streamKey, ok := parseRTMPStreamKey(url.Path)
		if !ok {
			glog.Errorf("Empty streamKey during RTMP authentication")
			return nil
		}
		manifestID, err := authenticateRTMPPublish(streamKey)
		if err != nil {
			glog.Errorf("RTMP authentication failed: %v", err)
			return nil
		}

		appData := CustomAppData{
			streamID:   generateUUID(),
			variantID:  generateUUID(), // TODO: multiple variants
			manifestID: manifestID,
		}
		return &appData
	}
}

func createRTMPStreamHandler(lpms *core.LPMS, rtmpStreams RTMPStreamMap, hlsMasters HLSMasterMap, hlsVariants HLSVariantMap) func(*url.URL, stream.RTMPVideoStream) error {
	return func(url *url.URL, rtmpStream stream.RTMPVideoStream) error {
		var hlsStream stream.HLSVideoStream
		appData, ok := rtmpStream.AppData().(*CustomAppData)
		if !ok {
			return errors.New("Mismatched app data type.")
		}
		manifest := stream.NewBasicHLSVideoManifest(appData.manifestID)

		// TODO: multiple variants
		hlsStream = stream.NewBasicHLSVideoStream(appData.variantID, hlsManifestLength)
		pl, err := hlsStream.GetStreamPlaylist()

		if err != nil {
			glog.Errorf("Error creating HLS stream playlist: %v", err)
			return err
		}

		err = manifest.AddVideoStream(hlsStream, &m3u8.Variant{URI: appData.variantID + ".m3u8", Chunklist: pl, VariantParams: m3u8.VariantParams{Bandwidth: 100}})
		if err != nil {
			glog.Errorf("Error adding variant to HLS manifest: %v", err)
			return err
		}

		opt := segmenter.SegmenterOptions{SegLength: 8 * time.Second}
		bgContext, cancel := context.WithCancel(context.Background())
		appData.cancelSegmenting = cancel

		// TODO: multiple variants
		hlsMasters[appData.manifestID] = HLSMasterManifest{
			manifest: manifest,
		}
		hlsVariants[appData.variantID] = HLSVariantData{
			hlsStream: hlsStream,
		}
		rtmpStreams[appData.streamID] = RTMPStreamData{
			rtmpStream: rtmpStream,
		}

		go func() {
			err := lpms.SegmentRTMPToHLS(bgContext, rtmpStream, hlsStream, opt)
			if err != nil {
				glog.Errorf("Error segmenting RTMP video stream: %v", err)
				rtmpStream.Close()
			}
		}()

		glog.Info("Publishing RTMP to http://localhost:7935/stream/", appData.manifestID, ".m3u8")
		glog.Info("Created manifest with ID ", appData.manifestID, " and variant ID ", appData.variantID)

		return nil
	}
}

func createRTMPStreamEndHandler(rtmpStreams RTMPStreamMap, hlsMasters HLSMasterMap, hlsVariants HLSVariantMap) func(*url.URL, stream.RTMPVideoStream) (err error) {
	return func(url *url.URL, rtmpStream stream.RTMPVideoStream) (err error) {
		appData, ok := rtmpStream.AppData().(*CustomAppData)
		if !ok {
			return errors.New("Mismatched app data type.")
		}
		glog.Info("RTMP stream ended.")
		appData.cancelSegmenting()
		delete(rtmpStreams, appData.streamID)
		delete(hlsMasters, appData.manifestID)
		delete(hlsVariants, appData.variantID) // TODO: multiple variants
		return nil
	}
}

func createRTMPPlayHandler(rtmpStreams RTMPStreamMap) func(*url.URL) (stream.RTMPVideoStream, error) {
	return func(url *url.URL) (stream.RTMPVideoStream, error) {
		streamID, ok := parseRTMPStreamKey(url.Path)
		if !ok {
			return nil, errors.New("Empty stream ID in RTMP playback.")
		}
		rtmpData, ok := rtmpStreams[streamID]
		if !ok {
			return nil, errors.New("No matching stream found for RTMP playback.")
		}
		return rtmpData.rtmpStream, nil
	}
}

func createHLSMasterPlaylistHandler(hlsMasters HLSMasterMap) func(*url.URL) (*m3u8.MasterPlaylist, error) {
	return func(url *url.URL) (*m3u8.MasterPlaylist, error) {
		manifestID, ok := parseHLSManifestID(url.Path)
		if !ok {
			glog.Info(url.Path)
			return nil, errors.New("Empty master manifest ID in HLS playback.")
		}
		hlsData, ok := hlsMasters[manifestID]
		if !ok {
			return nil, nil // allow fallthrough to media
		}

		manifest, err := hlsData.manifest.GetManifest() // TODO: actually return the master playlist
		return manifest, err
	}
}

func createHLSMediaPlaylistHandler(hlsVariants HLSVariantMap) func(*url.URL) (*m3u8.MediaPlaylist, error) {
	return func(url *url.URL) (*m3u8.MediaPlaylist, error) {
		variantID, ok := parseHLSManifestID(url.Path)
		if !ok {
			return nil, errors.New("Empty media manifest ID in HLS playback.")
		}
		hlsData, ok := hlsVariants[variantID]
		if !ok {
			return nil, errors.New("No matching stream found for HLS playback.")
		}

		// wait for HLSBuffer to be populated
		start := time.Now()
		for time.Since(start) < hlsWaitTime {
			pl, err := hlsData.hlsStream.GetStreamPlaylist()
			if err != nil || pl == nil || pl.Segments == nil || len(pl.Segments) <= 0 || pl.Segments[0] == nil || pl.Segments[0].URI == "" {
				if err == stream.ErrEOF {
					return nil, err
				}

				time.Sleep(time.Second)
				continue
			} else {
				return pl, nil
			}
		}
		return nil, errors.New("Error getting playlist")
	}
}

func createHLSSegmentHandler(hlsVariants HLSVariantMap) func(url *url.URL) ([]byte, error) {
	return func(url *url.URL) ([]byte, error) {
		manifestID, ok := parseHLSManifestID(url.Path)
		if !ok {
			return nil, errors.New("Could not parse manifest ID for HLS play.")
		}
		hlsData, ok := hlsVariants[manifestID]
		if !ok {
			return nil, errors.New("No matching stream found for HLS play.")
		}

		segmentName, ok := parseHLSSegmentName(url.Path)
		if !ok {
			return nil, errors.New("Empty segment name for HLS play.")
		}

		// TODO: parse out variant
		segment, err := hlsData.hlsStream.GetHLSSegment(segmentName)
		if err != nil {
			glog.Errorf("Error getting segment: %v", err)
			return nil, err
		}
		return segment.Data, nil
	}
}
