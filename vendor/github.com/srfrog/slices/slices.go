// Copyright (c) 2019 srfrog - https://srfrog.me
// Use of this source code is governed by the license in the LICENSE file.

// Package slices is a collection of functions to operate with string slices.
// Some functions were adapted from the strings package to work with slices, other
// were ported from PHP 'array_*' function equivalents.
package slices

import (
	"math/rand"
	"strings"
)

// ValueFunc is a value comparison func, used to compare element values in a slice.
type ValueFunc func(v string) bool

// Compare returns an integer comparing two slices lexicographically.
// The result will be 0 if a==b, or that a has all values of b.
// The result will be -1 if a < b, or a is shorter than b.
// The result will be +1 if a > b, or a is longer than b.
// A nil argument is equivalent to an empty slice.
func Compare(a, b []string) int {
	return CompareFunc(a, b, func(v1, v2 string) bool { return v1 == v2 })
}

// CompareFunc returns an integer comparing two slices with func f.
func CompareFunc(a, b []string, f func(string, string) bool) int {
	var i int

	m, n := len(a), len(b)
	switch {
	case m == 0:
		return -n
	case n == 0:
		return m
	case m > n:
		m = n
	}

	for i = 0; i < m; i++ {
		if !f(a[i], b[i]) {
			break
		}
	}

	return i - n
}

// Contains returns true if s is in a, false otherwise
func Contains(a []string, s string) bool {
	return Index(a, s) != -1
}

// ContainsAny returns true if any value in b is in a, false otherwise
func ContainsAny(a, b []string) bool {
	return IndexAny(a, b) != -1
}

// ContainsPrefix returns true if any element in a has prefix, false otherwise
func ContainsPrefix(a []string, prefix string) bool {
	return IndexFunc(a, ValueHasPrefix(prefix)) != -1
}

// ContainsSuffix returns true if any element in a has suffix, false otherwise
func ContainsSuffix(a []string, suffix string) bool {
	return IndexFunc(a, ValueHasSuffix(suffix)) != -1
}

// Count returns the number of occurrences of s in a.
func Count(a []string, s string) int {
	if len(a) == 0 {
		return 0
	}

	var n int

	for i := range a {
		if a[i] == s {
			n++
		}
	}

	return n
}

// Diff returns a slice with all the elements of b that are not found in a.
func Diff(a, b []string) []string {
	return DiffFunc(a, b, func(ss []string, v string) bool { return !Contains(ss, v) })
}

// DiffFunc compares the elements of b with those of b using f func.
// It returns a slice of the elements in b that are not found in a where cmp() == true.
func DiffFunc(a, b []string, f func([]string, string) bool) []string {
	var c []string

	for i := range a {
		if f(b, a[i]) {
			c = append(c, a[i])
		}
	}

	return c
}

// Equal returns a boolean reporting whether a and b are the same length and contain the
// same values, when compared lexicographically.
func Equal(a, b []string) bool {
	return len(a) == len(b) && Compare(a, b) == 0
}

// EqualFold returns a boolean reporting whether a and b
// are the same length and their values are equal under Unicode case-folding.
func EqualFold(a, b []string) bool {
	return len(a) == len(b) && CompareFunc(a, b, strings.EqualFold) == 0
}

// Fill is an alias of Repeat.
func Fill(n int, s string) []string {
	return Repeat(s, n)
}

// Filter returns a slice with all the elements of a that match string s.
func Filter(a []string, s string) []string {
	return FilterFunc(a, ValueEquals(s))
}

// FilterFunc returns a slice with all the elements of a that match string s that
// satisfy f(s). If func f returns true, the value will be filtered from b.
func FilterFunc(a []string, f ValueFunc) []string {
	if f == nil {
		return nil
	}

	var b []string

	for i := range a {
		if f(a[i]) {
			b = append(b, a[i])
		}
	}

	return b
}

// FilterPrefix returns a slice with all the elements of a that have prefix.
func FilterPrefix(a []string, prefix string) []string {
	return FilterFunc(a, ValueHasPrefix(prefix))
}

// FilterSuffix returns a slice with all the elements of a that have suffix.
func FilterSuffix(a []string, suffix string) []string {
	return FilterFunc(a, ValueHasSuffix(suffix))
}

// Chunk will divide a slice into subslices with size elements into a new 2d slice.
// The last chunk may contain less than size elements. If size less than 1, Chunk returns nil.
func Chunk(a []string, size int) [][]string {
	if size < 1 {
		return nil
	}

	aa := make([][]string, 0, (len(a)+size-1)/size)
	for size <= len(a) {
		a, aa = a[size:], append(aa, a[0:size:size])
	}

	if len(a) > 0 {
		aa = append(aa, a)
	}

	return aa
}

// Index returns the index of the first instance of s in a, or -1 if not found
func Index(a []string, s string) int {
	return IndexFunc(a, ValueEquals(s))
}

// IndexAny returns the index of the first instance of b in a, or -1 if not found
func IndexAny(a, b []string) int {
	ret, m := -1, 0

	for idx := range genIndex(a, b, Index) {
		if idx == m {
			return idx
		}
		if ret == -1 || idx < ret {
			ret = idx
		}
	}

	return ret
}

// IndexFunc returns the index of the first element in a where f(s) == true,
// or -1 if not found.
func IndexFunc(a []string, f ValueFunc) int {
	for i := range a {
		if f(a[i]) {
			return i
		}
	}

	return -1
}

// Intersect returns a slice with all the elements of b that are found in b.
func Intersect(a, b []string) []string {
	return DiffFunc(a, b, Contains)
}

// InsertAt inserts the values in slice a at index idx.
// This func will append the values if idx doesn't fit in the slice or is negative.
func InsertAt(a []string, idx int, values ...string) []string {
	m, n := len(a), len(values)
	if idx == -1 || idx > m {
		idx = m
	}

	if size := m + n; size <= cap(a) {
		b := a[:size]
		copy(b[idx+n:], a[idx:])
		copy(b[idx:], values)

		return b
	}

	b := make([]string, m+n)
	copy(b, a[:idx])
	copy(b[idx:], values)
	copy(b[idx+n:], a[idx:])

	return b
}

// LastIndex returns the index of the last instance of s in a, or -1 if not found
func LastIndex(a []string, s string) int {
	return LastIndexFunc(a, ValueEquals(s))
}

// LastIndexAny returns the index of the last instance of b in a, or -1 if not found
func LastIndexAny(a, b []string) int {
	ret, m := -1, len(a)

	for idx := range genIndex(a, b, LastIndex) {
		if idx == m {
			return idx
		}
		if idx != -1 && idx > ret {
			ret = idx
		}
	}

	return ret
}

// LastIndexFunc returns the index of the last element in a where f(s) == true,
// or -1 if not found.
func LastIndexFunc(a []string, f ValueFunc) int {
	for i := len(a) - 1; i >= 0; i-- {
		if f(a[i]) {
			return i
		}
	}

	return -1
}

// LastSearch returns the index of the last element containing substr in a,
// or -1 if not found. An empty substr matches any.
func LastSearch(a []string, substr string) int {
	return LastIndexFunc(a, ValueContains(substr))
}

// Map returns a new slice with the function 'mapping' applied to each element of b
func Map(mapping func(string) string, a []string) []string {
	if mapping == nil {
		return a
	}

	b := make([]string, len(a))

	for i := range a {
		b[i] = mapping(a[i])
	}

	return b
}

// Merge combines zero or many slices together, while preserving the order of elements.
func Merge(aa ...[]string) []string {
	var a []string

	for i := range aa {
		a = append(a, aa[i]...)
	}

	return a
}

// Pop removes the last element in a and returns it, shortening the slice by one.
// If a is empty returns empty string "".
// Note that this function will change the slice pointed by a.
func Pop(a *[]string) string {
	var s string

	if m := len(*a); m > 0 {
		s, *a = (*a)[m-1], (*a)[:m-1]
	}

	return s
}

// Push appends one or more values to a and returns the number of elements.
// Note that this function will change the slice pointed by a.
func Push(a *[]string, values ...string) int {
	if values != nil {
		*a = append(*a, values...)
	}

	return len(*a)
}

// Reduce applies the f func to each element in a and aggregates the result in acc
// and returns the total of the iterations. If there is only one value in the slice,
// it is returned.
// This func panics if f func is nil, or if the slice is empty.
func Reduce(a []string, f func(string, int, string) string) string {
	if f == nil {
		panic("slices: nil Reduce reducer func")
	}

	if len(a) == 0 {
		panic("slices: empty Reduce slice")
	}

	acc := a[0]

	Walk(a[1:], func(idx int, val string) {
		acc = f(acc, idx, val)
	})

	return acc
}

// Repeat returns a slice consisting of count copies of s.
func Repeat(s string, count int) []string {
	return RepeatFunc(func() string { return s }, count)
}

// RepeatFunc applies func f and returns a slice consisting of count values.
func RepeatFunc(f func() string, count int) []string {
	if count < 0 {
		panic("slices: negative Repeat count")
	}

	a := make([]string, count)

	for i := range a {
		a[i] = f()
	}

	return a
}

// Replace returns a copy of the slice a with the first n instances of old replaced by new.
// If n < 0, there is no limit on the number of replacements.
func Replace(a []string, old, new string, n int) []string {
	m := len(a)
	if old == new || m == 0 || n == 0 {
		return a
	}

	if m := Count(a, old); m == 0 {
		return a
	} else if n < 0 || m < n {
		n = m
	}

	t := append(a[:0:0], a...)
	for i := 0; i < m; i++ {
		if n == 0 {
			break
		}
		if t[i] == old {
			t[i] = new
			n--
		}
	}

	return t
}

// ReplaceAll returns a copy of the slice a with all instances of old replaced by new.
func ReplaceAll(a []string, old, new string) []string {
	return Replace(a, old, new, -1)
}

// Rand returns a new slice with n number of random elements of a
// using rand.Intn to select the elements.
// Note: You may want initialize the rand seed once in your program.
//
//    rand.Seed(time.Now().UnixNano())
//
func Rand(a []string, n int) []string {
	return RandFunc(a, n, rand.Intn)
}

// RandFunc returns a new slice with n number of random elements of a
// using func f to select the elements.
func RandFunc(a []string, n int, f func(int) int) []string {
	b := make([]string, n)
	if m := len(a); m > 0 {
		for i := 0; i < n; i++ {
			b[i] = a[f(m)]
		}
	}

	return b
}

// Reverse returns a slice of the reverse index order elements of a.
func Reverse(a []string) []string {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}

	return a
}

// Search returns the index of the first element containing substr in a,
// or -1 if not found. An empty substr matches any.
func Search(a []string, substr string) int {
	return IndexFunc(a, ValueContains(substr))
}

// Shift shifts the first element of a and returns it, shortening the slice by one.
// If a is empty returns empty string "".
// Note that this function will change the slice pointed by a.
func Shift(a *[]string) string {
	var s string

	if m := len(*a); m > 0 {
		s, *a = (*a)[0], (*a)[1:]
	}

	return s
}

// Shuffle returns a slice with randomized order of elements in a.
// Note: You may want initialize the rand seed once in your program.
//
//    rand.Seed(time.Now().UnixNano())
//
func Shuffle(a []string) []string {
	if m := len(a); m > 1 {
		rand.Shuffle(m, func(i, j int) {
			a[i], a[j] = a[j], a[i]
		})
	}

	return a
}

// Slice returns a subslice of the elements from the slice a as specified by the offset and length parameters.
//
// If offset > 0 the subslice will start at that offset in the slice.
// If offset < 0 the subslice will start that far from the end of the slice.
//
// If length > 0 then the subslice will have up to that many elements in it.
// If length == 0 then the subslice will begin from offset up to the end of the slice.
// If length < 0 then the subslice will stop that many elements from the end of the slice.
// If the slice a is shorter than the length, then only the available elements will be present.
//
// If the offset is larger than the size of the slice, an empty slice is returned.
func Slice(a []string, offset, length int) []string {
	m := len(a)
	if length == 0 {
		length = m
	}

	switch {
	case offset > m:
		return nil
	case offset < 0 && (m+offset) < 0:
		offset = 0
	case offset < 0:
		offset = m + offset
	}

	switch {
	case length < 0:
		length = m - offset + length
	case offset+length > m:
		length = m - offset
	}

	if length <= 0 {
		return nil
	}

	return a[offset : offset+length]
}

// Splice removes a portion of the slice a and replace it with the elements of another.
//
// If offset > 0 then the start of the removed portion is at that offset from the beginning of the slice.
// If offset < 0 then the start of the removed portion is at that offset from the end of the slice.
//
// If length > 0 then that many elements will be removed.
// If length == 0 no elements will be removed.
// If length == size removes everything from offset to the end of slice.
// If length < 0 then the end of the removed portion will be that many elements from the end of the slice.
//
// If b == nil then length elements are removed from a at offset.
// If b != nil then the elements are inserted at offset.
//
func Splice(a []string, offset, length int, b ...string) []string {
	m := len(a)
	switch {
	case offset > m:
		return a
	case offset < 0 && (m+offset) < 0:
		offset = 0
	case offset < 0:
		offset = m + offset
	}

	switch {
	case length < 0:
		length = m - offset + length
	case offset+length > m:
		length = m - offset
	}

	if length <= 0 {
		return a
	}

	return append(a[0:offset], append(b, a[offset+length:]...)...)
}

// split works almost like strings.genSplit() but for slices.
func split(a []string, sep string, n int) [][]string {
	switch {
	case n == 0:
		return nil
	case sep == "":
		return Chunk(a, 1)
	case n < 0:
		n = Count(a, sep) + 1
	}

	aa, i := make([][]string, n+1), 0
	for i < n {
		m := Index(a, sep)
		if m < 0 {
			break
		}
		aa[i] = a[:m]
		a = a[m+1:]
		i++
	}
	aa[i] = a

	return aa[:i+1]
}

// Split divides a slice a into subslices when any element matches the string sep.
//
// If a does not contain sep and sep is not empty, Split returns a
// 2d slice of length 1 whose only element is a.
//
// If sep is empty, Split returns a 2d slice of all elements in a.
// If a is nil and sep is empty, Split returns an empty slice.
//
// Split is akin to SplitN with a count of -1.
func Split(a []string, sep string) [][]string {
	return split(a, sep, -1)
}

// Split divides a slice a into subslices when n elements match the string sep.
//
// The count determines the number of subslices to return:
//   n > 0: at most n subslices; the last element will be the unsplit remainder.
//   n == 0: the result is nil (zero subslices)
//   n < 0: all subslices
//
// For other cases, see the documentation for Split.
func SplitN(a []string, sep string, n int) [][]string {
	return split(a, sep, n)
}

// Trim returns a slice with all the elements of a that don't match string s.
func Trim(a []string, s string) []string {
	return TrimFunc(a, ValueEquals(s))
}

// TrimFunc returns a slice with all the elements of a that don't match string s that
// satisfy f(s). If func f returns true, the value will be trimmed from a.
func TrimFunc(a []string, f ValueFunc) []string {
	if f == nil {
		return a
	}

	var b []string

	for i := range a {
		if !f(a[i]) {
			b = append(b, a[i])
		}
	}

	return b
}

// TrimPrefix returns a slice with all the elements of a that don't have prefix.
func TrimPrefix(a []string, prefix string) []string {
	return TrimFunc(a, ValueHasPrefix(prefix))
}

// TrimSuffix returns a slice with all the elements of a that don't have suffix.
func TrimSuffix(a []string, suffix string) []string {
	return TrimFunc(a, ValueHasSuffix(suffix))
}

// Unique returns a slice with duplicate values removed.
func Unique(a []string) []string {
	seen := make(map[string]struct{})

	b := a[:0]
	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			b = append(b, v)
		}
	}

	return b
}

// Unshift prepends one or more elements to *a and returns the number of elements.
// Note that this function will change the slice pointed by a
func Unshift(a *[]string, s ...string) int {
	if s != nil {
		*a = append(s, *a...)
	}

	return len(*a)
}

// ValueContains returns true if element value v contains substr.
func ValueContains(substr string) ValueFunc {
	return func(v string) bool {
		return strings.Contains(v, substr)
	}
}

// ValueEquals returns true if element value v equals s.
func ValueEquals(s string) ValueFunc {
	return func(v string) bool {
		return v == s
	}
}

// ValueHasPrefix returns true if element value begins with prefix.
func ValueHasPrefix(prefix string) ValueFunc {
	return func(v string) bool {
		return strings.HasPrefix(v, prefix)
	}
}

// ValueHasPrefix returns true if element value ends with suffix.
func ValueHasSuffix(suffix string) ValueFunc {
	return func(v string) bool {
		return strings.HasSuffix(v, suffix)
	}
}

// Walk applies the f func to each element in a.
func Walk(a []string, f func(idx int, val string)) {
	for idx := range a {
		f(idx, a[idx])
	}
}

func genIndex(a, b []string, f func([]string, string) int) <-chan int {
	l := len(b)

	rc := make(chan int, l)
	go func() {
		defer close(rc)

		if l == 0 {
			rc <- -1
			return
		}

		for i := 0; i < l; i++ {
			rc <- f(a, b[i])
		}
	}()

	return rc
}
