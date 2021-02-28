package config

import (
	"errors"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watch will watch the config files and reload the
// config data whenever one of the files is created,
// or changes.
func Watch() error { return c.Watch() }

// Watch will watch the config files and reload the
// config data whenever one of the files is created,
// or changes.
func (c *Config) Watch() error {
	var mu sync.Mutex
	return c.updated(func(e fsnotify.Event) {
		mu.Lock()
		defer mu.Unlock()

		raw, err := ioutil.ReadFile(e.Name)
		if err != nil {
			log.Println("config.Watch:", err)
			return
		}
		tmp := copyVal(c.elem)

		err = c.unmarshal(raw, c.config)
		if err != nil {
			log.Println("config.Watch:", err)
			return
		}

		err = merge(c.elem, tmp)
		if err != nil {
			log.Println("config.Watch:", err)
			return
		}
	})
}

// Updated will return a channel which will never close and will
// recieve an empty struct every time a config file is created,
// or written to.
func Updated() (<-chan struct{}, error) {
	return c.Updated()
}

// Updated will return a channel which will never close and will
// recieve an empty struct every time a config file is created,
// or written to.
func (c *Config) Updated() (<-chan struct{}, error) {
	ch := make(chan struct{})
	return ch, c.updated(func(e fsnotify.Event) {
		ch <- struct{}{}
	})
}

func (c *Config) updated(f func(fsnotify.Event)) error {
	var (
		err error
	)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				// if the channel is closed, just return
				if !ok {
					return
				}
				switch event.Op {
				case fsnotify.Write, fsnotify.Create:
					f(event)
				default:
					continue
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					continue
				}
				if err != nil {
					log.Println("config watcher error:", err)
				}
			}
		}
	}()

	n := 0
	for _, path := range c.paths {
		for _, file := range c.files {
			f := filepath.Join(path, file)
			err = watcher.Add(f)
			if err != nil {
				return err
			}
			n++
		}
	}
	if n == 0 {
		return errors.New("not watching any config files")
	}
	return nil
}
