package vedio

import (
	"reflect"
	"testing"
)

type benchDependency struct{}

func (benchDependency) Init() (*benchDependency, error) {
	return &benchDependency{}, nil
}

type benchService struct {
	dep benchDependency
}

func (s benchService) Init(dep benchDependency) (*benchService, error) {
	s.dep = dep
	return &s, nil
}

func benchmarkReset() {
	mut.Lock()
	container = make(map[reflect.Type]map[string]resolver)
	mut.Unlock()

	AllowDuplicateRegistration = false
	AllowReadWithNoLock = false
}

func benchmarkSetup(b *testing.B, lifecycle LifeCycle) {
	b.Helper()

	benchmarkReset()

	if err := Register[benchDependency](); err != nil {
		b.Fatal(err)
	}

	if err := Register[benchService](WithLifeCycle(lifecycle)); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkResolve_Singleton(b *testing.B) {
	benchmarkSetup(b, SINGLETON)

	if _, err := Resolve[benchService](); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		if _, err := Resolve[benchService](); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolve_Transient(b *testing.B) {
	benchmarkSetup(b, TRANSIENT)

	b.ReportAllocs()

	for b.Loop() {
		if _, err := Resolve[benchService](); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolve_Scoped(b *testing.B) {
	benchmarkSetup(b, SCOPED)

	scope := NewScope()
	defer scope.Close()

	if _, err := Resolve[benchService](WithScope(scope)); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		if _, err := Resolve[benchService](WithScope(scope)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolve_Locked(b *testing.B) {
	benchmarkSetup(b, SINGLETON)

	if _, err := Resolve[benchService](); err != nil {
		b.Fatal(err)
	}

	AllowReadWithNoLock = false

	b.ReportAllocs()

	for b.Loop() {
		if _, err := Resolve[benchService](); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolve_NoLock(b *testing.B) {
	benchmarkSetup(b, SINGLETON)

	if _, err := Resolve[benchService](); err != nil {
		b.Fatal(err)
	}

	AllowReadWithNoLock = true
	defer func() {
		AllowReadWithNoLock = false
	}()

	b.ReportAllocs()

	for b.Loop() {
		if _, err := Resolve[benchService](); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolve_Parallel(b *testing.B) {
	benchmarkSetup(b, SINGLETON)

	if _, err := Resolve[benchService](); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := Resolve[benchService](); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkNewScope(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		NewScope()
	}
}

func BenchmarkScopeClose(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		scope := NewScope()
		scope.Close()
	}
}
