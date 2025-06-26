package utils

import (
	"bytes"
	"compress/gzip"
	"danmu-core/logger"
	"encoding/base64"
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"strconv"
)

// GenerateMsToken 生成随机的msToken
func GenerateMsToken(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+="
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b) + "=_"
}

// GzipCompressAndBase64Encode 将数据进行gzip压缩并进行Base64编码
func GzipCompressAndBase64Encode(data []byte) (string, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	if _, err := w.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

func RandomUserAgent() string {
	osList := []string{
		"(Windows NT 10.0; WOW64)", "(Windows NT 10.0; Win64; x64)",
		"(Windows NT 6.3; WOW64)", "(Windows NT 6.3; Win64; x64)",
		"(Windows NT 6.1; Win64; x64)", "(Windows NT 6.1; WOW64)",
		"(X11; Linux x86_64)",
		"(Macintosh; Intel Mac OS X 10_12_6)",
	}

	chromeVersionList := []string{
		"110.0.5481.77", "110.0.5481.30", "109.0.5414.74", "108.0.5359.71",
		"108.0.5359.22", "98.0.4758.48", "97.0.4692.71",
	}

	os := osList[rand.Intn(len(osList))]
	chromeVersion := chromeVersionList[rand.Intn(len(chromeVersionList))]

	return fmt.Sprintf("Mozilla/5.0 %s AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", os, chromeVersion)
}

func GetUserUniqueID() string {
	id := rand.Int63n(7999999999999999999-7300000000000000000+1) + 7300000000000000000
	return strconv.FormatInt(id, 10)
}

func SafeRun(f func()) {
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()

			// 获取文件名和行号
			_, file, line, ok := runtime.Caller(2)
			callerInfo := "unknown"
			if ok {
				callerInfo = fmt.Sprintf("%s:%d", file, line)
			}

			logger.Error().
				Str("caller", callerInfo).
				Interface("panic", err).
				Str("stack", string(stack)).
				Msg("Panic recovered in SafeRun")
		}
	}()
	f()
}

func NormalizeTimestamp(timestamp int64) int64 {
	// 转换为字符串，用于判断位数
	tsStr := strconv.FormatInt(timestamp, 10)

	switch len(tsStr) {
	case 10: // 秒级时间戳
		return timestamp * 1000
	case 13: // 毫秒级时间戳
		return timestamp
	default:
		return timestamp
	}
}
