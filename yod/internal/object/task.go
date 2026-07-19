package object

import "sync"

// SpawnTask — מריץ פונקציית יוד ב-goroutine ברקע ומחזיר Task. מוגן במנעול הריצה
// כדי לא לרוץ במקביל לקוד יוד אחר (UI/משימות) — משותף למפרש ולמכונה.
func SpawnTask(fn Object) Object {
	switch fn.(type) {
	case *Function, *Closure, *CompiledFunction:
	default:
		return &Error{Message: "משימה מצפה לפונקציה"}
	}
	t := &Task{Ch: make(chan Object, 1)}
	go func() {
		LockYod()
		res := Call(fn)
		UnlockYod()
		t.Ch <- res
	}()
	return t
}

// AwaitTask — ממתין לסיום משימה ומחזיר את תוצאתה. משחרר את מנעול הריצה בזמן ההמתנה.
func AwaitTask(obj Object) Object {
	t, ok := obj.(*Task)
	if !ok {
		return &Error{Message: "המתן מצפה לערך מסוג משימה"}
	}
	if t.Done {
		return t.Val
	}
	var val Object
	WithoutYodLock(func() {
		val = <-t.Ch
	})
	t.Val = val
	t.Done = true
	return val
}

// ParallelTasks — ממתין לרשימת משימות ומחזיר רשימת תוצאות. משחרר את המנעול בזמן ההמתנה.
func ParallelTasks(obj Object) Object {
	arr, ok := obj.(*Array)
	if !ok {
		return &Error{Message: "במקביל מצפה לרשימה"}
	}
	out := make([]Object, len(arr.Elements))
	WithoutYodLock(func() {
		var wg sync.WaitGroup
		for i, el := range arr.Elements {
			wg.Add(1)
			go func(i int, el Object) {
				defer wg.Done()
				out[i] = AwaitTask(el)
			}(i, el)
		}
		wg.Wait()
	})
	return &Array{Elements: out}
}
