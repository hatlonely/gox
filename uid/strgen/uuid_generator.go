package strgen

import (
	"encoding/hex"
	"github.com/google/uuid"
)

type UUIDOptions struct {
	Version    string
	WithHyphens bool // 是否包含中划线连字符，默认不包含
}

type UUIDGenerator struct {
	version     string
	withHyphens bool
}

func NewUUIDGeneratorWithOptions(options *UUIDOptions) *UUIDGenerator {
	if options == nil {
		options = &UUIDOptions{Version: "v4"}
	}
	if options.Version == "" {
		options.Version = "v4"
	}
	
	return &UUIDGenerator{
		version:     options.Version,
		withHyphens: options.WithHyphens,
	}
}

func (g *UUIDGenerator) Generate() string {
	var u uuid.UUID
	switch g.version {
	case "v1":
		u, _ = uuid.NewUUID()
	case "v4":
		u = uuid.New()
	case "v6":
		u = uuid.Must(uuid.NewV6())
	case "v7":
		u = uuid.Must(uuid.NewV7())
	default:
		u = uuid.New()
	}
	
	if g.withHyphens {
		return u.String()
	}
	
	// 直接将字节转换为十六进制字符串，避免字符串替换
	return hex.EncodeToString(u[:])
}