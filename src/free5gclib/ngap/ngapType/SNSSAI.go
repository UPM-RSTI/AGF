package ngapType

// Need to import "free5gclib/aper" if it uses "aper"

type SNSSAI struct {
	SST          SST
	SD           *SD                                     `aper:"optional"`
	IEExtensions *ProtocolExtensionContainerSNSSAIExtIEs `aper:"optional"`
}
