package domain

type FileType string

const (
	FileTypeNone FileType = "" //arquivo físico
	FileTypePDF  FileType = "PDF"
	FileTypeEPUB FileType = "EPUB"
	FileTypeTXT  FileType = "TXT"
)

func (f FileType) IsValid() bool {
	switch f {
	case FileTypeNone, FileTypePDF, FileTypeEPUB, FileTypeTXT:
		return true
	}
	return false
}
