package ngapType

// Need to import "free5gclib/aper" if it uses "aper"

/* Sequence of = 35, FULL Name = struct PDUSessionResourceReleasedListNot */
/* PDUSessionResourceReleasedItemNot */
type PDUSessionResourceReleasedListNot struct {
	List []PDUSessionResourceReleasedItemNot `aper:"valueExt,sizeLB:1,sizeUB:256"`
}
