package async

import "github.com/lithictech/aperitif/mariobros"

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
