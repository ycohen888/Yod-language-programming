package stdlib

import "yod/internal/object"

// NewBasisModule — מודול בסיס (הדפס/קלט/המרות זמינים גם גלובלית).
func NewBasisModule() *object.Module {
	m := &object.Module{Name: "בסיס", Attrs: map[string]object.Object{}}
	m.Attrs["גרסה"] = &object.String{Value: "0.50.6"}
	m.Attrs["הסבר"] = &object.String{Value: "הדפס, קלט, אורך, הוסף, למספר, למחרוזת, סוג, טווח, אקראי — זמינים גלובלית; וגם/או/לא"}
	return m
}
