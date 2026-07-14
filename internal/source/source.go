package source

type Position struct {
	Offset int
	Line   int
	Col    int
}

type Range struct {
	Start Position
	End   Position
}

func (r Range) IsEmpty() bool {
	return r.Start.Offset >= r.End.Offset
}

func (r Range) Contains(offset int) bool {
	return offset >= r.Start.Offset && offset < r.End.Offset
}
