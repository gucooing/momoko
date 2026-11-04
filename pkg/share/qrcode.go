package share

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/url"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

// 公开分享页路由（与前端 /public/share/:token 一致）。
const publicSharePath = "/public/share/"

// qrSize 是二维码图片边长（像素，不含留白）；qrQuiet 为四周留白。
const (
	qrSize  = 320
	qrQuiet = 24
)

// PublicLink 由站点源（scheme://host[:port]）与令牌拼出公开分享链接；origin 必须是 http(s) 源，不能带路径。
func PublicLink(origin, token string) (string, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(origin), "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("站点地址不合法")
	}
	return u.Scheme + "://" + u.Host + publicSharePath + token, nil
}

// QRCode 生成公开分享链接及其 PNG 二维码（纠错等级 M，白底黑码带留白，便于手机扫码）。
func QRCode(origin, token string) (string, []byte, error) {
	link, err := PublicLink(origin, token)
	if err != nil {
		return "", nil, err
	}
	code, err := qr.Encode(link, qr.M, qr.Auto)
	if err != nil {
		return "", nil, err
	}
	scaled, err := barcode.Scale(code, qrSize, qrSize)
	if err != nil {
		return "", nil, err
	}
	canvas := image.NewRGBA(image.Rect(0, 0, qrSize+2*qrQuiet, qrSize+2*qrQuiet))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(qrQuiet, qrQuiet, qrQuiet+qrSize, qrQuiet+qrSize), scaled, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return "", nil, err
	}
	return link, buf.Bytes(), nil
}
