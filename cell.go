package rtk

type Cell struct {
	Character  string // Extended Grapheme Cluster
	Foreground Color
	Background Color
	Attribute  AttributeMask
}
