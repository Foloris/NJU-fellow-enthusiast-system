package service

import (
	"strings"
	"unicode/utf16"

	"campus-fellow/server/internal/model"
)

// Match browser/TypeScript string length, including surrogate pairs for emoji.
func length(s string, min, max int) bool {
	n := len(utf16.Encode([]rune(s)))
	return n >= min && n <= max
}
func optionalLength(s *string, max int) bool { return s == nil || length(*s, 0, max) }
func trimPointer(s **string) {
	if *s != nil {
		v := strings.TrimSpace(**s)
		*s = &v
	}
}
func nullEmpty(s **string) {
	trimPointer(s)
	if *s != nil && **s == "" {
		*s = nil
	}
}
func contains(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

func validateExchange(in *model.ExchangeInput) error {
	if in.Kind == "" {
		in.Kind = "share"
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Content = strings.TrimSpace(in.Content)
	nullEmpty(&in.ImageURL)
	if !contains(in.Kind, "share", "news") {
		return fail(400, "无效的内容类型")
	}
	if !length(in.Title, 4, 100) {
		return fail(400, "标题应为4—100个字符")
	}
	if !length(in.Content, 10, 10000) {
		return fail(400, "正文应为10—10000个字符")
	}
	if !optionalLength(in.ImageURL, 500) {
		return fail(400, "图片地址过长")
	}
	return nil
}
func validateMate(in *model.MateInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Details = strings.TrimSpace(in.Details)
	in.Contact = strings.TrimSpace(in.Contact)
	in.Duration = strings.TrimSpace(in.Duration)
	nullEmpty(&in.ImageURL)
	nullEmpty(&in.Location)
	nullEmpty(&in.Notes)
	if !length(in.Title, 4, 100) || !length(in.Details, 10, 10000) {
		return fail(400, "标题应为4—100个字符，详细说明应为10—10000个字符")
	}
	if len(in.Tags) < 1 || len(in.Tags) > 6 {
		return fail(400, "请选择1—6个标签")
	}
	for i := range in.Tags {
		in.Tags[i] = strings.TrimSpace(in.Tags[i])
		if !length(in.Tags[i], 1, 16) {
			return fail(400, "标签应为1—16个字符")
		}
	}
	if !length(in.Contact, 2, 100) || !length(in.Duration, 1, 100) {
		return fail(400, "请填写联系方式和持续时间，且均不超过100个字符")
	}
	if !optionalLength(in.ImageURL, 500) || !optionalLength(in.Location, 100) || !optionalLength(in.Notes, 1000) {
		return fail(400, "图片地址、地点或备注过长")
	}
	return nil
}
