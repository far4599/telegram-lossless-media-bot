package model

type DocType int

const (
	DocTypeUnknown DocType = iota
	DocTypePhoto
	DocTypeVideo
)

func (dt DocType) String() string {
	switch dt {
	case DocTypePhoto:
		return "photo"
	case DocTypeVideo:
		return "video"
	default:
		return "unknown"
	}
}
