// Copyright 2022 Gustavo C. Viegas. All rights reserved.

package vk

import (
	"testing"

	"github.com/gviegas/scene/driver"
)

func TestCmdBuffer(t *testing.T) {
	zcb := cmdBuffer{}
	call := "tDrv.NewCmdBuffer()"
	// NewCmdBuffer.
	if cb, err := tDrv.NewCmdBuffer(); err == nil {
		if cb == nil {
			t.Errorf("%s\nhave nil, nil\nwant non-nil, nil", call)
			return
		}
		cb := cb.(*cmdBuffer)
		if cb.d != &tDrv {
			t.Errorf("%s: cb.d\nhave %p\nwant %p", call, cb.d, &tDrv)
		}
		if cb.pool == zcb.pool {
			t.Errorf("%s: cb.pool\nhave %v\nwant valid handle", call, cb.pool)
		}
		if cb.cb == nil {
			t.Errorf("%s: cb.cb\nhave nil\nwant non-nil", call)
		}
		// Destroy.
		cb.Destroy()
		if cb.d != nil || cb.pool != zcb.pool || cb.cb != nil {
			t.Errorf("cb.Destroy(): cb\nhave %v\nwant %v", cb, cmdBuffer{})
		}
	} else if cb != nil {
		t.Errorf("%s\nhave %p, %v\nwant nil, %v", call, cb, err, err)
	}
}

func TestCmdRecording(t *testing.T) {
	cb, err := tDrv.NewCmdBuffer()
	if err != nil {
		t.Error("NewCmdBuffer failed, cannot test command recording")
		return
	}
	defer cb.Destroy()
	src, err := tDrv.NewBuffer(1024, true, 0)
	if err != nil {
		t.Error("NewBuffer failed, cannot test command recording")
		return
	}
	defer src.Destroy()
	dst, err := tDrv.NewBuffer(769, true, 0)
	if err != nil {
		t.Error("NewBuffer failed, cannot test command recording")
		return
	}
	defer dst.Destroy()
	if err = cb.Begin(); err != nil {
		t.Errorf("(error) cb.Begin(): %v", err)
		return
	}
	cb.Fill(src, 16, 0x2a, 256)
	cb.Barrier([]driver.Barrier{
		{
			SyncBefore:   driver.SCopy,
			SyncAfter:    driver.SCopy,
			AccessBefore: driver.ACopyWrite,
			AccessAfter:  driver.ACopyRead | driver.ACopyWrite,
		},
	})
	cb.CopyBuffer(&driver.BufferCopy{
		From:    src,
		FromOff: 0,
		To:      dst,
		ToOff:   512,
		Size:    256,
	})
	err = cb.End()
	if err != nil {
		t.Errorf("(error) cb.End(): %v", err)
		return
	}
	ch := make(chan error)
	go tDrv.Commit([]driver.CmdBuffer{cb}, ch)
	err = <-ch
	if err != nil {
		t.Errorf("(error) tDrv.Commit(): %v", err)
	} else {
		t.Log(src.Bytes())
		t.Log(dst.Bytes())
	}
	cb.Reset()
}
