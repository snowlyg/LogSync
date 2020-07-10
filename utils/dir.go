package utils

// bis 床旁
// nis 护理白板
// nws 护士站主机
// webapp 门旁
type DirName int

const (
	OTHER DirName = iota
	NIS
	BIS
	WEBAPP
	NWS
)

func (d DirName) String() string {
	switch d {
	case BIS:
		return "bis"
	case NIS:
		return "nis"
	case NWS:
		return "nws"
	case WEBAPP:
		return "webapp"
	default:
		return "other"
	}
}
