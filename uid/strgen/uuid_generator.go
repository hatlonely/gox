package strgen

import (
	"strings"
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
	var uuidStr string
	switch g.version {
	case "v1":
		v1, _ := uuid.NewUUID()
		uuidStr = v1.String()
	case "v4":
		uuidStr = uuid.New().String()
	case "v6":
		uuidStr = uuid.Must(uuid.NewV6()).String()
	case "v7":
		uuidStr = uuid.Must(uuid.NewV7()).String()
	default:
		uuidStr = uuid.New().String()
	}
	
	if !g.withHyphens {
		return strings.ReplaceAll(uuidStr, "-", "")
	}
	return uuidStr
}