package ngapType

// Need to import "free5gclib/aper" if it uses "aper"

/* Sequence of = 35, FULL Name = struct PDUSessionResourceAdmittedList */
/* PDUSessionResourceAdmittedItem */
type PDUSessionResourceAdmittedList struct {
	List []PDUSessionResourceAdmittedItem `aper:"valueExt,sizeLB:1,sizeUB:256"`
}
