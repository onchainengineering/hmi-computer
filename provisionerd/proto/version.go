package proto

import (
	"github.com/coder/coder/v2/apiversion"
)

const (
	CurrentMajor = 1
	CurrentMinor = 1
)

var CurrentVersion = apiversion.New(CurrentMajor, CurrentMinor)
