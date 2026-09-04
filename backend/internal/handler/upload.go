package handler

import (
	crand "crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"qsl-management/pkg/response"
)

const maxImageSize = 5 << 20 // 5MB

// UploadImage 后台上传图片（卡片版本等），保存到 ./uploads 并返回访问路径
func UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, 400, "请选择图片文件")
		return
	}
	if file.Size > maxImageSize {
		response.Fail(c, 400, "图片不能超过 5MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		response.Fail(c, 400, "仅支持 jpg/jpeg/png/gif/webp 格式")
		return
	}

	// 内容嗅探：防止扩展名伪装
	f, err := file.Open()
	if err != nil {
		response.Fail(c, 400, "读取文件失败")
		return
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := io.ReadFull(f, buf)
	if n > 0 && !strings.HasPrefix(http.DetectContentType(buf[:n]), "image/") {
		response.Fail(c, 400, "文件内容不是图片")
		return
	}

	b := make([]byte, 12)
	if _, err := crand.Read(b); err != nil {
		response.Error(c, "生成文件名失败")
		return
	}
	name := hex.EncodeToString(b) + ext
	if err := os.MkdirAll("uploads", 0755); err != nil {
		response.Error(c, "创建上传目录失败")
		return
	}
	if err := c.SaveUploadedFile(file, filepath.Join("uploads", name)); err != nil {
		response.Error(c, "保存文件失败")
		return
	}
	response.OK(c, gin.H{"url": "/uploads/" + name})
}
