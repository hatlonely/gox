package parser

import (
	"strings"
)

type SeparatorLineParserOptions struct {
	Separator string `cfg:"separator" def:"\t"`
}

type SeparatorLineParser struct {
	separator string
}

func NewSeparatorLineParserWithOptions(options *SeparatorLineParserOptions) (*SeparatorLineParser, error) {
	return &SeparatorLineParser{
		separator: options.Separator,
	}, nil
}

func (p *SeparatorLineParser) Parse(line string) (ChangeType, string, string, error) {
	parts := strings.Split(line, p.separator)
	
	if len(parts) < 2 {
		return ChangeTypeUnknown, "", "", nil
	}
	
	key := parts[0]
	value := parts[1]
	changeType := ChangeTypeAdd
	
	if len(parts) >= 3 && parts[2] != "" {
		switch strings.ToLower(parts[2]) {
		case "add":
			changeType = ChangeTypeAdd
		case "update":
			changeType = ChangeTypeUpdate
		case "delete":
			changeType = ChangeTypeDelete
		default:
			changeType = ChangeTypeAdd
		}
	}
	
	return changeType, key, value, nil
}
