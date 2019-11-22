package client

type Type uint8

const (
	TypeFile Type = iota
	TypeFolder
)

type File struct {
	Type Type
	ID   string
	Name string
	Size string
	Date string
}

type Share struct {
	UserID     int    `json:"userid"`
	FolderID   int    `json:"folder_id"`
	FileChk    string `json:"file_chk"`
	FolderName string `json:"folder_name"`
	FolderTime string `json:"folder_time"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Url        string `json:"url"`
	PageTitle  string `json:"page_title"`

	Files []*File
}
