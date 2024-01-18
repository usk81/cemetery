// go:build go1.20
package toki

func (t Toki) Compare(u Toki) int {
	return t.Time.Compare(u.Time)
}
