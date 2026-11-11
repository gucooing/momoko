package themeasset

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	stddraw "image/draw"
	"image/jpeg"
	"image/png"

	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"
	xwebp "golang.org/x/image/webp"
)

const (
	// MaxImageSize 上传 / 主题包内图片的原始大小上限。
	MaxImageSize = 10 << 20
	// MaxFontSize 字体大小上限。
	MaxFontSize = 4 << 20
	// MaxImageSide 转换后图片最长边。
	MaxImageSide = 2560
	// maxImagePixels 解码前按图片头拒绝超大尺寸，防止解压炸弹占满内存。
	maxImagePixels = 50_000_000
	webpQuality    = 80
	// webpMethod 编码速度 / 体积折中：4 比 2 慢约 3 倍、体积只小约 5%（纯 Go 编码器）
	webpMethod = 2
)

var (
	ErrImageType     = errors.New("图片仅支持 JPG、PNG、WebP 格式")
	ErrImageTooLarge = errors.New("图片大小不能超过 10 MB")
	ErrImagePixels   = errors.New("图片尺寸过大（最多 5000 万像素）")
	ErrImageDecode   = errors.New("图片无法解析，文件可能已损坏")
	ErrImageEncode   = errors.New("图片转换失败")
	ErrFontType      = errors.New("字体仅支持 woff2 格式")
	ErrFontTooLarge  = errors.New("字体大小不能超过 4 MB")
)

// Processed 处理后的资源。
type Processed struct {
	Kind string
	Ext  string
	Data []byte
	// 图片宽高（缩放后）
	Width, Height int
	OriginalSize  int64
}

type imageFormat int

const (
	formatUnknown imageFormat = iota
	formatJPEG
	formatPNG
	formatWebP
)

func sniffImage(data []byte) imageFormat {
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return formatJPEG
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		return formatPNG
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return formatWebP
	default:
		return formatUnknown
	}
}

// ProcessImage 校验并转换图片：按内容识别 JPG / PNG / WebP，最长边缩到 2560 以内，
// 按 JPEG 的 EXIF 方向摆正，再编码为有损 WebP（质量 80）。
func ProcessImage(raw []byte) (*Processed, error) {
	if len(raw) > MaxImageSize {
		return nil, ErrImageTooLarge
	}
	format := sniffImage(raw)
	var (
		cfg image.Config
		err error
	)
	switch format {
	case formatJPEG:
		cfg, err = jpeg.DecodeConfig(bytes.NewReader(raw))
	case formatPNG:
		cfg, err = png.DecodeConfig(bytes.NewReader(raw))
	case formatWebP:
		cfg, err = xwebp.DecodeConfig(bytes.NewReader(raw))
	default:
		return nil, ErrImageType
	}
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, ErrImageDecode
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return nil, ErrImagePixels
	}

	var src image.Image
	switch format {
	case formatJPEG:
		src, err = jpeg.Decode(bytes.NewReader(raw))
	case formatPNG:
		src, err = png.Decode(bytes.NewReader(raw))
	case formatWebP:
		src, err = xwebp.Decode(bytes.NewReader(raw))
	}
	if err != nil {
		return nil, ErrImageDecode
	}

	sb := src.Bounds()
	w, h := fitSize(sb.Dx(), sb.Dy(), MaxImageSide)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if w == sb.Dx() && h == sb.Dy() {
		stddraw.Draw(dst, dst.Bounds(), src, sb.Min, stddraw.Src)
	} else {
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, sb, xdraw.Src, nil)
	}
	if format == formatJPEG {
		dst = applyOrientation(dst, jpegOrientation(raw))
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, unpremultiply(dst), webp.Options{Quality: webpQuality, Method: webpMethod}); err != nil {
		return nil, ErrImageEncode
	}
	out := buf.Bytes()
	// 本来就是尺寸合适的静态 WebP（不含 EXIF / XMP）且比重新编码更小时保留原图，避免反复导入导出越转越大
	if format == formatWebP && w == sb.Dx() && h == sb.Dy() {
		if plain, ok := plainWebP(raw); ok && len(plain) <= len(out) {
			out = plain
		}
	}
	b := dst.Bounds()
	return &Processed{
		Kind:         KindImage,
		Ext:          ExtWebP,
		Data:         out,
		Width:        b.Dx(),
		Height:       b.Dy(),
		OriginalSize: int64(len(raw)),
	}, nil
}

// plainWebP 判断是否为不含动画 / EXIF / XMP 的静态 WebP，返回截掉 RIFF 之后多余字节的内容。
func plainWebP(raw []byte) ([]byte, bool) {
	if len(raw) < 30 {
		return nil, false
	}
	total := uint64(binary.LittleEndian.Uint32(raw[4:8])) + 8
	if total < 30 || total > uint64(len(raw)) {
		return nil, false
	}
	data := raw[:total]
	switch string(data[12:16]) {
	case "VP8 ", "VP8L":
		return data, true
	case "VP8X":
		const animation, xmp, exif = 0x02, 0x04, 0x08
		if data[20]&(animation|xmp|exif) != 0 {
			return nil, false
		}
		return data, true
	}
	return nil, false
}

// ProcessFont 校验字体：只接受 woff2（魔数 wOF2），原样保存。
func ProcessFont(raw []byte) (*Processed, error) {
	if len(raw) > MaxFontSize {
		return nil, ErrFontTooLarge
	}
	if len(raw) < 48 || !bytes.Equal(raw[:4], []byte("wOF2")) {
		return nil, ErrFontType
	}
	return &Processed{
		Kind:         KindFont,
		Ext:          ExtWOFF2,
		Data:         raw,
		OriginalSize: int64(len(raw)),
	}, nil
}

// fitSize 等比缩放到最长边不超过 limit。
func fitSize(w, h, limit int) (int, int) {
	if w <= limit && h <= limit {
		return w, h
	}
	if w >= h {
		nh := int(int64(h) * int64(limit) / int64(w))
		return limit, max(nh, 1)
	}
	nw := int(int64(w) * int64(limit) / int64(h))
	return max(nw, 1), limit
}

// unpremultiply 把预乘 alpha 的 RGBA 转为 NRGBA（完全不透明时直接复用像素）。
func unpremultiply(img *image.RGBA) *image.NRGBA {
	pix := img.Pix
	for i := 3; i < len(pix); i += 4 {
		a := pix[i]
		if a == 0xff {
			continue
		}
		if a == 0 {
			pix[i-3], pix[i-2], pix[i-1] = 0, 0, 0
			continue
		}
		pix[i-3] = uint8(min(int(pix[i-3])*0xff/int(a), 0xff))
		pix[i-2] = uint8(min(int(pix[i-2])*0xff/int(a), 0xff))
		pix[i-1] = uint8(min(int(pix[i-1])*0xff/int(a), 0xff))
	}
	return &image.NRGBA{Pix: pix, Stride: img.Stride, Rect: img.Rect}
}

// jpegOrientation 读取 JPEG APP1 EXIF 里的方向标记（1 - 8），没有或无法解析返回 1。
func jpegOrientation(data []byte) int {
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xff {
			return 1
		}
		marker := data[i+1]
		if marker == 0xd8 || (marker >= 0xd0 && marker <= 0xd7) || marker == 0x01 {
			i += 2
			continue
		}
		if marker == 0xda || marker == 0xd9 {
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		seg := data[i+4 : i+2+size]
		if marker == 0xe1 && len(seg) > 6 && string(seg[:6]) == "Exif\x00\x00" {
			return exifOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 1
}

func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	if bo.Uint16(tiff[2:4]) != 42 {
		return 1
	}
	off := int(bo.Uint32(tiff[4:8]))
	if off < 8 || off+2 > len(tiff) {
		return 1
	}
	n := int(bo.Uint16(tiff[off : off+2]))
	for k := 0; k < n; k++ {
		e := off + 2 + k*12
		if e+12 > len(tiff) {
			return 1
		}
		if bo.Uint16(tiff[e:e+2]) == 0x0112 {
			v := int(bo.Uint16(tiff[e+8 : e+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation 按 EXIF 方向把图片摆正。
func applyOrientation(img *image.RGBA, orientation int) *image.RGBA {
	if orientation <= 1 || orientation > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if orientation >= 5 {
		dw, dh = h, w
	}
	out := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orientation {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			si := img.PixOffset(b.Min.X+x, b.Min.Y+y)
			di := out.PixOffset(dx, dy)
			copy(out.Pix[di:di+4], img.Pix[si:si+4])
		}
	}
	return out
}
