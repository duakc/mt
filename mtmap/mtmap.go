package mtmap

import "github.com/duakc/mt"

func MergeMap[K comparable, V any](mm ...map[K]V) map[K]V {
	allLen := mt.Sum(mt.Map(mm, func(s map[K]V) int {
		return len(mm)
	}))
	res := make(map[K]V, allLen)
	for _, m := range mm {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}
