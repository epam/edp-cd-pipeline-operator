package error

type CISNotFoundError string

func (j CISNotFoundError) Error() string {
	return string(j)
}
