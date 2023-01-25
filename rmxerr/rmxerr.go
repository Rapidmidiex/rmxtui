package rmxerr

type (
	ErrMsg struct {
		Err error
	}
)

func (m ErrMsg) Error() string {
	return m.Err.Error()
}
