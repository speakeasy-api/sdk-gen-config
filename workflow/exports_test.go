package workflow

func SetRandStringBytesFunc(f func(n int) string) {
	randStringBytes = f
}
