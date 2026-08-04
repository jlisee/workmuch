package backend

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/screensaver"
	"github.com/jezek/xgb/xproto"
)

const maxX11PropertyLength = ^uint32(0)

type xgbLinuxX11Client struct {
	conn             *xgb.Conn
	root             xproto.Window
	activeWindowAtom xproto.Atom
	netWMNameAtom    xproto.Atom
}

func connectLinuxX11(display string) (linuxX11Client, error) {
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		return nil, err
	}

	closeWithError := func(err error) (linuxX11Client, error) {
		conn.Close()
		return nil, err
	}

	if err := screensaver.Init(conn); err != nil {
		return closeWithError(fmt.Errorf("initialize XScreenSaver extension: %w", err))
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	if screen == nil {
		return closeWithError(errors.New("X11 display has no default screen"))
	}

	activeWindowAtom, err := internExistingAtom(conn, "_NET_ACTIVE_WINDOW")
	if err != nil {
		return closeWithError(fmt.Errorf("find _NET_ACTIVE_WINDOW atom: %w", err))
	}
	netWMNameAtom, err := internExistingAtom(conn, "_NET_WM_NAME")
	if err != nil {
		return closeWithError(fmt.Errorf("find _NET_WM_NAME atom: %w", err))
	}

	return &xgbLinuxX11Client{
		conn:             conn,
		root:             screen.Root,
		activeWindowAtom: activeWindowAtom,
		netWMNameAtom:    netWMNameAtom,
	}, nil
}

func internExistingAtom(conn *xgb.Conn, name string) (xproto.Atom, error) {
	reply, err := xproto.InternAtom(conn, true, uint16(len(name)), name).Reply()
	if err != nil {
		return xproto.AtomNone, err
	}
	return reply.Atom, nil
}

func (c *xgbLinuxX11Client) ActiveWindow() (uint32, error) {
	if c.activeWindowAtom != xproto.AtomNone {
		reply, err := c.property(c.root, c.activeWindowAtom)
		if err != nil {
			return 0, err
		}
		if reply.Format == 32 && len(reply.Value) >= 4 {
			window := xproto.Window(xgb.Get32(reply.Value))
			if window != xproto.WindowNone {
				return uint32(window), nil
			}
		}
	}

	focus, err := xproto.GetInputFocus(c.conn).Reply()
	if err != nil {
		return 0, err
	}
	if focus.Focus == xproto.WindowNone || focus.Focus == xproto.Window(xproto.InputFocusPointerRoot) {
		return 0, errors.New("X11 input focus does not identify a window")
	}

	return findTopLevelWindow(
		uint32(focus.Focus),
		uint32(c.root),
		func(window uint32) (uint32, error) {
			tree, err := xproto.QueryTree(c.conn, xproto.Window(window)).Reply()
			if err != nil {
				return 0, err
			}
			return uint32(tree.Parent), nil
		},
		c.WindowTitle,
	)
}

func (c *xgbLinuxX11Client) WindowTitle(window uint32) (string, error) {
	if c.netWMNameAtom != xproto.AtomNone {
		title, err := c.textProperty(xproto.Window(window), c.netWMNameAtom)
		if err != nil {
			return "", err
		}
		if title != "" {
			return title, nil
		}
	}

	return c.textProperty(xproto.Window(window), xproto.AtomWmName)
}

func (c *xgbLinuxX11Client) ProgramName(window uint32) (string, error) {
	reply, err := c.property(xproto.Window(window), xproto.AtomWmClass)
	if err != nil {
		return "", err
	}
	return parseWMClass(reply.Value), nil
}

func (c *xgbLinuxX11Client) IdleMilliseconds() (uint32, error) {
	reply, err := screensaver.QueryInfo(c.conn, xproto.Drawable(c.root)).Reply()
	if err != nil {
		return 0, err
	}
	return reply.MsSinceUserInput, nil
}

func (c *xgbLinuxX11Client) Close() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	return nil
}

func (c *xgbLinuxX11Client) textProperty(window xproto.Window, atom xproto.Atom) (string, error) {
	reply, err := c.property(window, atom)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(reply.Value), "\x00"), nil
}

func (c *xgbLinuxX11Client) property(window xproto.Window, atom xproto.Atom) (*xproto.GetPropertyReply, error) {
	return xproto.GetProperty(
		c.conn,
		false,
		window,
		atom,
		xproto.GetPropertyTypeAny,
		0,
		maxX11PropertyLength,
	).Reply()
}
