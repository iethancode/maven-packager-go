package maven

import "os"

func statOrNil(p string) (os.FileInfo, error) {
	return os.Stat(p)
}
