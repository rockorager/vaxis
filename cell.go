package rtk

type Cell struct {
	EGC        string // Extended Grapheme Cluster
	Foreground Color
	Background Color
	Attribute  AttributeMask
}
