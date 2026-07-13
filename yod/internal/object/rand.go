package object

import (
	"math/rand"
	"sync"
	"time"
)

var (
	randMu  sync.Mutex
	randSrc = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func randFloat() float64 {
	randMu.Lock()
	defer randMu.Unlock()
	return randSrc.Float64()
}

func randInt63n(n int64) int64 {
	randMu.Lock()
	defer randMu.Unlock()
	return randSrc.Int63n(n)
}

func shuffleObjects(elems []Object) {
	randMu.Lock()
	defer randMu.Unlock()
	randSrc.Shuffle(len(elems), func(i, j int) {
		elems[i], elems[j] = elems[j], elems[i]
	})
}
