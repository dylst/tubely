package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(mediaType string) string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic("failed to generate random bytes")
	}
	filename := base64.RawURLEncoding.EncodeToString(b)
	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", filename, ext)
}

func (cfg apiConfig) getObjectURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
}

func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	type output struct{
		Streams []struct{
			CodecType  string `json:"codec_type"`
			Width int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	res := output{}
	err = json.Unmarshal(buf.Bytes(), &res)
	if err != nil {
		return "", err
	}

	ind := -1 
	for i, stream := range res.Streams {
		if stream.CodecType == "video" {
			ind = i
			break
		}
	}

	if ind == -1 {
		return "", errors.New("No video found")
	}

	aspectRatio := calculateAspectRatio(res.Streams[ind].Width, res.Streams[ind].Height)

	return aspectRatio, nil
}

func calculateAspectRatio(width, height int) string {
	ratio := float64(width) / float64(height)
	target16_9 := 16.0 / 9.0 
	target9_16 := 9.0 / 16.0 
	epsilon := 0.05

	if math.Abs(ratio - target16_9) < epsilon {
		return "landscape"
	} else if math.Abs(ratio - target9_16) < epsilon {
		return "portrait"
	}

	return "other"
}