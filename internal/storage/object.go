package storage

type UploadedFile struct {
	Provider string
	Key      string
	URL      string
	Name     string
}

func (f UploadedFile) Keys() (r2, gcs, s3 string) {
	switch f.Provider {
	case "r2":
		return f.Key, "", ""
	case "gcs", "native":
		return "", f.Key, ""
	default:
		return "", "", f.Key
	}
}
