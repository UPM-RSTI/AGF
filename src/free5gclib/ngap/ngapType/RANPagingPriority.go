package ngapType

// Need to import "free5gclib/aper" if it uses "aper"

type RANPagingPriority struct {
	Value int64 `aper:"valueLB:1,valueUB:256"`
}
