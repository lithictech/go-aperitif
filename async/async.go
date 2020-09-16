package async

import "github.com/lithictech/go-aperitif/mariobros"

type Goer func(name string, f func())

func Sync(_ string, f func()) {
	f()
}

func Async(name string, f func()) {
	go func() {
		mario := mariobros.Yo(name)
		defer mario()
		f()
	}()
}

func NewSpying(g Goer) *Spying {
	return &Spying{goer: g, CallCount: 0}
}

type Spying struct {
	goer      Goer
	CallCount int
	Calls     []string
}

func (s *Spying) Go(name string, f func()) {
	s.CallCount++
	s.Calls = append(s.Calls, name)
	f()
}
