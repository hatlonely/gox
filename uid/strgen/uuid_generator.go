package strgen

import "github.com/google/uuid"

type UUIDOptions struct {
	Version string
}

type UUIDGenerator struct {
	version string
}

func NewUUIDGeneratorWithOptions(options *UUIDOptions) *UUIDGenerator {
	if options == nil {
		options = &UUIDOptions{Version: "v4"}
	}
	if options.Version == "" {
		options.Version = "v4"
	}
	
	return &UUIDGenerator{
		version: options.Version,
	}
}

func (g *UUIDGenerator) Generate() string {
	switch g.version {
	case "v1":
		v1, _ := uuid.NewUUID()
		return v1.String()
	case "v4":
		return uuid.New().String()
	case "v6":
		return uuid.Must(uuid.NewV6()).String()
	case "v7":
		return uuid.Must(uuid.NewV7()).String()
	default:
		return uuid.New().String()
	}
}