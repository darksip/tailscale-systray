package sysmenu

// base structure for menu abstraction

import "fmt"

type EvtHnd func()

type Melt struct {
	Id        string
	Label     string
	Handler   EvtHnd
	Disabled  bool
	Hidden    bool
	Separator bool
	Defawlt   bool
}

type SysMenu struct {
	Items           []Melt
	hideCallback    func(id string, value bool)
	disableCallBack func(id string, value bool)
	addCallback     func(e Melt)
	setHndCallback  func(id string, e EvtHnd)
	setLabel        func(id string, label string)
	setIcon         func(id string, iconame string)
}

func (sm *SysMenu) GetById(id string) (*Melt, error) {
	for i := range sm.Items {
		if sm.Items[i].Id == id {
			return &sm.Items[i], nil
		}
	}
	return nil, fmt.Errorf("item with ID %q not found", id)
}

func NewSysMenu(hideCB func(id string, v bool), disableCB func(id string, v bool),
	add func(e Melt), hndCB func(id string, e EvtHnd),
	lbl func(id string, l string), setico func(id string, iconame string)) *SysMenu {
	sm := &SysMenu{
		Items:           []Melt{},
		hideCallback:    hideCB,
		disableCallBack: disableCB,
		addCallback:     add,
		setHndCallback:  hndCB,
		setLabel:        lbl,
		setIcon:         setico,
	}
	return sm
}

func (sm *SysMenu) SetHiddenAll(ids []string, v bool) {
	for _, id := range ids {
		sm.SetHidden(id, v)
	}
}

func (sm *SysMenu) SetHidden(id string, v bool) {
	for i := range sm.Items {
		if sm.Items[i].Id == id {
			if sm.Items[i].Hidden != v {
				sm.Items[i].Hidden = v
				sm.hideCallback(id, v)
			}
		}
	}
}

func (sm *SysMenu) SetDisabled(id string, v bool) {
	for i := range sm.Items {
		if sm.Items[i].Id == id {
			if sm.Items[i].Disabled != v {
				sm.Items[i].Disabled = v
				sm.disableCallBack(id, v)
			}
		}
	}
}

func (sm *SysMenu) Add(e Melt) {
	sm.Items = append(sm.Items, e)
	sm.addCallback(e)
}

func (sm *SysMenu) SetHandler(id string, handler EvtHnd) {
	for i := range sm.Items {
		if sm.Items[i].Id == id {
			sm.Items[i].Handler = handler
			sm.setHndCallback(id, handler)
		}
	}
}

func (sm *SysMenu) SetLabel(id string, label string) {
	for i := range sm.Items {
		if sm.Items[i].Id == id {
			sm.Items[i].Label = label
			sm.setLabel(id, label)
		}
	}
}

func (sm *SysMenu) SetIcon(id string, iconame string) {
	sm.setIcon(id, iconame)
}
