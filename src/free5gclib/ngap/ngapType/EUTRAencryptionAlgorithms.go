package ngapType

import "free5gclib/aper"

// Need to import "free5gclib/aper" if it uses "aper"

type EUTRAencryptionAlgorithms struct {
	Value aper.BitString `aper:"sizeExt,sizeLB:16,sizeUB:16"`
}
