package storage

type UploadedFile struct {
	Provider	string
	Key		string
	URL		string
	Name		string
}

func (file UploadedFile) Keys() (r2, gcs, s3 string) {
	switch file.Provider {
	case "r2":
		return file.Key, "", ""
	case "gcs", "native":
		return "", file.Key, ""
	default:
		return "", "", file.Key
	}
}
