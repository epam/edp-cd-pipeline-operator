package error

type CISNotFound string

func (j CISNotFound) Error() string {
	return string(j)
}
