//go:build windows

package stdlib

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/lxn/win"

	"yod/internal/object"
)

var (
	regionClassRegistered bool
	regionPickResult      *object.Hash
	regionCancelled       bool
	regionDone            bool
	regionDragging        bool
	regionStartX          int32
	regionStartY          int32
	regionCurX            int32
	regionCurY            int32
	regionHwnd            win.HWND
	regionWndProcCB       uintptr

	gdi32Region         = syscall.NewLazyDLL("gdi32.dll")
	procCreatePenRegion = gdi32Region.NewProc("CreatePen")
	procRectangleRegion = gdi32Region.NewProc("Rectangle")
)

const (
	regionClassName = "YodRecordRegionPicker"
	vkEscapeRegion  = 0x1B
	vkReturnRegion  = 0x0D
	psSolidRegion   = 0
)

func recordPickRegion(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("הקלטה.בחר_אזור מצפה ל־0 ארגומנטים")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	regionPickResult = nil
	regionCancelled = false
	regionDone = false
	regionDragging = false
	regionStartX, regionStartY, regionCurX, regionCurY = 0, 0, 0, 0

	if err := ensureRegionClass(); err != nil {
		return errObj("הקלטה.בחר_אזור: " + err.Error())
	}

	sw := int32(win.GetSystemMetrics(win.SM_CXSCREEN))
	sh := int32(win.GetSystemMetrics(win.SM_CYSCREEN))

	hwnd := win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_LAYERED|win.WS_EX_TOOLWINDOW,
		syscall.StringToUTF16Ptr(regionClassName),
		syscall.StringToUTF16Ptr("בחירת אזור הקלטה"),
		win.WS_POPUP|win.WS_VISIBLE,
		0, 0, sw, sh,
		0, 0, win.GetModuleHandle(nil), nil,
	)
	if hwnd == 0 {
		return errObj("הקלטה.בחר_אזור: יצירת חלון נכשלה")
	}
	regionHwnd = hwnd
	setWindowAlpha(hwnd, 90)

	win.SetCursor(win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_CROSS)))
	win.SetForegroundWindow(hwnd)

	var msg win.MSG
	for !regionDone {
		ret := win.GetMessage(&msg, 0, 0, 0)
		if ret == -1 {
			break
		}
		if ret == 0 {
			// WM_QUIT של האפליקציה — מחזירים לתור כדי לא לבלוע סגירה אמיתית
			win.PostQuitMessage(int32(msg.WParam))
			regionCancelled = true
			break
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}

	if regionHwnd != 0 {
		win.DestroyWindow(regionHwnd)
		regionHwnd = 0
	}

	if regionCancelled || regionPickResult == nil {
		return object.Nil
	}
	return regionPickResult
}

func ensureRegionClass() error {
	if regionClassRegistered {
		return nil
	}
	regionWndProcCB = syscall.NewCallback(regionWndProc)
	var wc win.WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = regionWndProcCB
	wc.HInstance = win.GetModuleHandle(nil)
	wc.HCursor = win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_CROSS))
	wc.HbrBackground = win.HBRUSH(win.GetStockObject(win.BLACK_BRUSH))
	wc.LpszClassName = syscall.StringToUTF16Ptr(regionClassName)
	if atom := win.RegisterClassEx(&wc); atom == 0 {
		errCode := win.GetLastError()
		if errCode != 1410 { // ERROR_CLASS_ALREADY_EXISTS
			return fmt.Errorf("RegisterClassEx נכשל (קוד %d)", errCode)
		}
	}
	regionClassRegistered = true
	return nil
}

func endRegionPick() {
	regionDone = true
	if regionHwnd != 0 {
		h := regionHwnd
		regionHwnd = 0
		win.DestroyWindow(h)
	}
}

func regionWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_LBUTTONDOWN:
		regionDragging = true
		regionStartX = int32(int16(lParam & 0xFFFF))
		regionStartY = int32(int16((lParam >> 16) & 0xFFFF))
		regionCurX, regionCurY = regionStartX, regionStartY
		win.SetCapture(hwnd)
		win.InvalidateRect(hwnd, nil, true)
		return 0
	case win.WM_MOUSEMOVE:
		if regionDragging {
			regionCurX = int32(int16(lParam & 0xFFFF))
			regionCurY = int32(int16((lParam >> 16) & 0xFFFF))
			win.InvalidateRect(hwnd, nil, true)
		}
		return 0
	case win.WM_LBUTTONUP:
		if regionDragging {
			regionDragging = false
			win.ReleaseCapture()
			regionCurX = int32(int16(lParam & 0xFFFF))
			regionCurY = int32(int16((lParam >> 16) & 0xFFFF))
			finishRegionPick()
		}
		return 0
	case win.WM_KEYDOWN:
		vk := uint32(wParam)
		if vk == vkEscapeRegion {
			regionCancelled = true
			endRegionPick()
			return 0
		}
		if vk == vkReturnRegion && regionDragging {
			regionDragging = false
			win.ReleaseCapture()
			finishRegionPick()
			return 0
		}
		return 0
	case win.WM_PAINT:
		var ps win.PAINTSTRUCT
		hdc := win.BeginPaint(hwnd, &ps)
		if regionDragging || (regionStartX != regionCurX || regionStartY != regionCurY) {
			x1, y1, x2, y2 := normalizeRect(regionStartX, regionStartY, regionCurX, regionCurY)
			color := uintptr(win.RGB(61, 214, 198))
			pen, _, _ := procCreatePenRegion.Call(uintptr(psSolidRegion), 2, color)
			oldPen := win.SelectObject(hdc, win.HGDIOBJ(pen))
			oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))
			_, _, _ = procRectangleRegion.Call(uintptr(hdc), uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2))
			win.SelectObject(hdc, oldBrush)
			win.SelectObject(hdc, oldPen)
			if pen != 0 {
				win.DeleteObject(win.HGDIOBJ(pen))
			}
		}
		win.EndPaint(hwnd, &ps)
		return 0
	case win.WM_DESTROY:
		regionDone = true
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func normalizeRect(x1, y1, x2, y2 int32) (int32, int32, int32, int32) {
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	return x1, y1, x2, y2
}

func finishRegionPick() {
	x1, y1, x2, y2 := normalizeRect(regionStartX, regionStartY, regionCurX, regionCurY)
	w := x2 - x1
	h := y2 - y1
	if w < 2 || h < 2 {
		regionCancelled = true
		endRegionPick()
		return
	}
	hwnd := regionHwnd
	var pt win.POINT
	pt.X, pt.Y = x1, y1
	if hwnd != 0 {
		win.ClientToScreen(hwnd, &pt)
	}
	sx, sy := pt.X, pt.Y

	regionPickResult = &object.Hash{Pairs: map[string]object.Object{
		"x":    &object.Number{Value: float64(sx)},
		"y":    &object.Number{Value: float64(sy)},
		"רוחב": &object.Number{Value: float64(w)},
		"גובה": &object.Number{Value: float64(h)},
	}}
	endRegionPick()
}
