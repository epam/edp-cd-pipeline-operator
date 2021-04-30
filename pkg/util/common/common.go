package common

func GetStringP(val string) *string {
	return &val
}

func Int64Ptr(i int64) *int64 {
	return &i
}
