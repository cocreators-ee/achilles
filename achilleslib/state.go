package achilleslib

import (
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/progress"
)

type state int

const (
	searchingFiles = iota
	scanningFiles
	done
)

var (
	mapMutex     = sync.Mutex{}
	messageMutex = sync.Mutex{}
	GlobalModel  *Model
	finished     = false
)

func InitialModel() Model {
	return Model{
		State:           searchingFiles,
		ScannedLibs:     &atomic.Int32{},
		ScannedBins:     &atomic.Int32{},
		LoadingProgress: 0.0,
		Libs:            map[string]*atomic.Int32{},
		Bins:            []string{},
		Messages:        []string{},
		Progress:        progress.New(progress.WithDefaultGradient()),
		Table:           getTable(),
	}
}

func LibsLen(m Model) int {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	return len(m.Libs)
}
