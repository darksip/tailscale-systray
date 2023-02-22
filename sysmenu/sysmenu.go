package sysmenu

import "fmt"

type evtHnd func()

type mElt struct {
	id        string
	label     string
	handler   evtHnd
	enabled   bool
	hidden    bool
	separator bool
	defawlt   bool
}

type SysMenu struct {
	items          []mElt
	hideCallback   func(id string, value bool)
	enableCallBack func(id string, value bool)
}

func (sm *SysMenu) getById(id string) (*mElt, error) {
	for i := range sm.items {
		if sm.items[i].id == id {
			return &sm.items[i], nil
		}
	}
	return nil, fmt.Errorf("item with ID %q not found", id)
}

func NewSysMenu(hideCB func(id string, v bool), enableCB func(id string, v bool)) *SysMenu {
	sm := &SysMenu{
		items:          []mElt{},
		hideCallback:   hideCB,
		enableCallBack: enableCB,
	}
	return sm
}

func (sm *SysMenu) Hide(id string) {
	for i := range sm.items {
		if sm.items[i].id == id {

		}
	}
}
