package platform

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"danmu-core/generated/dystruct"
	"encoding/hex"
	"fmt"
	"github.com/elliotchance/orderedmap"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	webRidRg = regexp.MustCompile(`douyin\.com/.*?(\d+)(?:\?.*)?$`)
)

func getTTWID() (string, error) {
	res, err := http.Get("https://live.douyin.com/")
	if err != nil {
		return "", fmt.Errorf("获取直播 URL 失败: %w", err)
	}

	for _, cookie := range res.Cookies() {
		if cookie.Name == "ttwid" {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("未找到 ttwid cookie")
}

func (dy *Douyin) BuildRequestURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	queryParams := parsedURL.Query()

	queryParams.Set("aid", "6383")
	queryParams.Set("device_platform", "web")
	queryParams.Set("browser_language", "zh-CN")
	queryParams.Set("browser_platform", "Win32")

	if dy.ua != "" {
		uaParts := strings.Split(dy.ua, "/")
		if len(uaParts) > 0 {
			browserName := uaParts[0]
			queryParams.Set("browser_name", browserName)

			splitByName := strings.Split(dy.ua, browserName)
			if len(splitByName) > 1 {
				lastPart := splitByName[len(splitByName)-1]
				if len(lastPart) > 1 { // 确保至少有2个字符才能去掉第一个
					browserVersion := lastPart[1:]
					queryParams.Set("browser_version", browserVersion)
				}
			}
		}
	}

	parsedURL.RawQuery = queryParams.Encode()
	return parsedURL.String(), nil
}

func (dy *Douyin) decompressGzip(data []byte) ([]byte, error) {
	buf := dy.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		dy.bufferPool.Put(buf)
	}()

	buf.Write(data)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	result := bytes.NewBuffer(make([]byte, 0, len(data)*2))
	if _, err = io.Copy(result, gz); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func HasGzipEncoding(headers []*dystruct.Webcast_Im_PushHeader) bool {

	for _, header := range headers {
		if header.Key == "compress_type" && header.Value == "gzip" {
			return true
		}
	}
	return false
}

func BuildRequestURL(ua string, rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	queryParams := parsedURL.Query()

	queryParams.Set("aid", "6383")
	queryParams.Set("device_platform", "web")
	queryParams.Set("browser_language", "zh-CN")
	queryParams.Set("browser_platform", "Win32")

	browserName := strings.Split(ua, "/")[0]
	queryParams.Set("browser_name", browserName)

	tempSplit := strings.Split(ua, browserName)
	browserVersion := ""
	if len(tempSplit) > 1 {
		browserVersion = strings.TrimPrefix(tempSplit[len(tempSplit)-1], "/")
	}
	queryParams.Set("browser_version", browserVersion)

	parsedURL.RawQuery = queryParams.Encode()

	return parsedURL.String(), nil
}

// NewSigMap 创建一个有序的map
func NewSigMap(roomID, uniqueId string) *orderedmap.OrderedMap {
	smap := orderedmap.NewOrderedMap()
	smap.Set("live_id", "1")
	smap.Set("aid", "6383")
	smap.Set("version_code", "180800")
	smap.Set("webcast_sdk_version", "1.0.14-beta.0")
	smap.Set("room_id", roomID)
	smap.Set("sub_room_id", "")
	smap.Set("sub_channel_id", "")
	smap.Set("did_rule", "3")
	smap.Set("user_unique_id", uniqueId)
	smap.Set("device_platform", "web")
	smap.Set("device_type", "")
	smap.Set("ac", "")
	smap.Set("identity", "audience")
	return smap
}

func NewWebCast5Param(roomId, uniqueId, signature string) url.Values {
	webcast5Params := url.Values{}
	webcast5Params.Set("room_id", roomId)
	webcast5Params.Set("compress", "gzip")
	webcast5Params.Set("version_code", "180800")
	webcast5Params.Set("webcast_sdk_version", "1.0.14-beta.0")
	webcast5Params.Set("live_id", "1")
	webcast5Params.Set("did_rule", "3")
	webcast5Params.Set("user_unique_id", uniqueId)
	webcast5Params.Set("identity", "audience")
	webcast5Params.Set("signature", signature)
	return webcast5Params
}

// GetxMSStub 拼接map并返回其MD5哈希值的十六进制字符串
func GetxMSStub(params *orderedmap.OrderedMap) string {
	var sigParams strings.Builder
	for i, key := range params.Keys() {
		if i > 0 {
			sigParams.WriteString(",")
		}
		value, _ := params.Get(key)
		sigParams.WriteString(fmt.Sprintf("%s=%s", key, value))
	}
	hash := md5.Sum([]byte(sigParams.String()))
	return hex.EncodeToString(hash[:])
}
