package inhibit

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	dbusScreenSaver = "org.freedesktop.ScreenSaver"
	dbusPath        = "/org/freedesktop/ScreenSaver"
	dbusInterface   = "org.freedesktop.ScreenSaver"
)

var ErrNotInitialized = errors.New("inhibitor not initialized")

type Inhibitor struct {
	conn   *dbus.Conn
	cookie uint32
	active bool
}

func New() (*Inhibitor, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to session bus: %w", err)
	}
	return &Inhibitor{conn: conn}, nil
}

func (i *Inhibitor) Inhibit() error {
	if i == nil || i.conn == nil {
		return ErrNotInitialized
	}
	if i.active {
		return nil
	}
	obj := i.conn.Object(dbusScreenSaver, dbusPath)
	call := obj.Call(dbusInterface+".Inhibit", 0, "pmusic", "Music playback")
	if call.Err != nil {
		return fmt.Errorf("inhibit screensaver: %w", call.Err)
	}
	if err := call.Store(&i.cookie); err != nil {
		return fmt.Errorf("store inhibit cookie: %w", err)
	}
	i.active = true
	return nil
}

func (i *Inhibitor) UnInhibit() error {
	if i == nil || i.conn == nil {
		return ErrNotInitialized
	}
	if !i.active {
		return nil
	}
	obj := i.conn.Object(dbusScreenSaver, dbusPath)
	call := obj.Call(dbusInterface+".UnInhibit", 0, i.cookie)
	if call.Err != nil {
		return fmt.Errorf("uninhibit screensaver: %w", call.Err)
	}
	i.cookie = 0
	i.active = false
	return nil
}

func (i *Inhibitor) Close() error {
	if i == nil || i.conn == nil {
		return nil
	}
	_ = i.UnInhibit()
	return i.conn.Close()
}
