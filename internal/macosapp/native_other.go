//go:build !darwin || !cgo

package macosapp

type unsupportedMainAppService struct{}

func NewMainAppService() LoginItem {
	return unsupportedMainAppService{}
}

func (unsupportedMainAppService) Status() (LoginItemState, error) {
	return LoginItemUnsupported, nil
}

func (unsupportedMainAppService) Register() error {
	return nil
}

type unsupportedMoveDialog struct{}

func NewMoveDialog() MoveDialog {
	return unsupportedMoveDialog{}
}

func (unsupportedMoveDialog) Show() error {
	return nil
}
