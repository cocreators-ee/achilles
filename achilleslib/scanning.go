package achilleslib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/zaolin/u-root/pkg/ldd"
)

var (
	libDirs = []string{"/lib", "/lib32", "/usr/lib", "/usr/lib32", "/usr/local/lib", "/usr/local/lib32"}
	binDirs = []string{"/bin", "/sbin", "/usr/bin", "/usr/sbin", "/usr/local/bin", "/usr/local/sbin"}
)

var (
	scanned      = map[string]bool{}
	scannedMutex = sync.Mutex{}
)

func (m *Model) resolveAbsolute(path string) (string, error) {
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return path, nil
}

func (m *Model) searchFiles(searchPath string, files chan string, filesWg *sync.WaitGroup, filterFn func(string, os.DirEntry) bool) {
	defer filesWg.Done()

	searchPath, err := m.resolveAbsolute(searchPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			m.addMessage(err.Error())
		}
		return
	}

	err = filepath.WalkDir(searchPath, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		filePath, err = m.resolveAbsolute(filePath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				m.addMessage(fmt.Sprintf("%s: %s", filePath, err.Error()))
			}
			return nil
		}

		if filePath == searchPath {
			return nil
		}

		if !entry.IsDir() {
			if filterFn(filePath, entry) {
				files <- filePath
			}
		}

		return nil
	})
	if err != nil {
		m.addMessage(err.Error())
	}
}

func (m *Model) findLibs() (chan string, *sync.WaitGroup) {
	files := make(chan string)
	filesWg := new(sync.WaitGroup)

	for _, libDir := range libDirs {
		filesWg.Add(1)
		go m.searchFiles(libDir, files, filesWg, func(path string, entry os.DirEntry) bool {
			ext := filepath.Ext(path)
			if ext == ".so" {
				return true
			}
			return strings.Contains(ext, ".so.")
		})
	}

	go func() {
		filesWg.Wait()
		close(files)
	}()

	return files, filesWg
}

func (m *Model) findBins() (chan string, *sync.WaitGroup) {
	files := make(chan string)
	filesWg := new(sync.WaitGroup)

	for _, binDir := range binDirs {
		filesWg.Add(1)
		go m.searchFiles(binDir, files, filesWg, func(path string, entry os.DirEntry) bool {
			info, err := entry.Info()
			if err != nil {
				return false
			}
			mode := info.Mode()
			return (mode.Perm() & 0o111) > 0
		})
	}

	go func() {
		filesWg.Wait()
		close(files)
	}()

	return files, filesWg
}

func (m *Model) scanFile(scanPath string) {
	scannedMutex.Lock()
	_, alreadyScanned := scanned[scanPath]
	scannedMutex.Unlock()
	if alreadyScanned {
		return
	}

	if strings.Contains(scanPath, " ") {
		return
	}

	// TODO: This ldd lib seems less than awesome, and we might want to support other systems too
	infoList, err := ldd.Ldd([]string{scanPath})
	if err != nil {
		return
	}

	for _, info := range infoList {
		libPath, err := m.resolveAbsolute(info.FullName)
		if err != nil {
			continue
		}

		mapMutex.Lock()
		_, ok := m.Libs[libPath]
		if !ok {
			m.Libs[libPath] = &atomic.Int32{}
			m.ScannedLibs.Add(1)
		}
		m.Libs[libPath].Add(1)
		mapMutex.Unlock()
	}
}

func (m *Model) scanFiles(libs chan string, bins chan string) {
	scanWg := new(sync.WaitGroup)
	scanWg.Add(2)

	go func() {
		libsWg := new(sync.WaitGroup)
		for lib := range libs {
			libsWg.Add(1)

			mapMutex.Lock()
			_, ok := m.Libs[lib]
			if !ok {
				go func(lib string) {
					m.scanFile(lib)
					libsWg.Done()
				}(lib)
			}
			mapMutex.Unlock()
		}
		libsWg.Wait()
		scanWg.Done()
	}()

	go func() {
		binsWg := new(sync.WaitGroup)
		for bin := range bins {
			binsWg.Add(1)
			m.Bins = append(m.Bins, bin)
			go func(bin string) {
				m.scanFile(bin)
				m.ScannedBins.Add(1)
				binsWg.Done()
			}(bin)
		}
		binsWg.Wait()
		scanWg.Done()
	}()

	scanWg.Wait()
	m.State = done
	m.Table = getTable()
}

func (m *Model) FindFiles() {
	m.State = searchingFiles
	libsChan, libsWg := m.findLibs()
	binsChan, binsWg := m.findBins()

	go func() {
		libsWg.Wait()
		binsWg.Wait()
		m.State = scanningFiles
	}()

	m.scanFiles(libsChan, binsChan)
}
