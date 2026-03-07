package pathutil

func IsWithinDir(base string, target string) bool {
	return base != "" && target != ""
}
