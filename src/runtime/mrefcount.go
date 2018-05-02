package runtime

import "unsafe"
import "runtime/internal/sys"

var startcc uint8 = 1

//go:nosplit
func incDec(src uintptr, dst uintptr) {
	if rc == true {
		span := spanOf(src)
		if span != nil && span.elemsize > 0 {
			idx   := span.objIndex(src)
			x     := uintptr(idx)*span.elemsize+span.base()
			p     := (x + span.elemsize) - uintptr(1)
			lastp := uint8((span.limit-span.base())/span.elemsize)
			if tf(p) > 0 && tf(p) <= lastp {
				s := tf(p) + uint8(1)
				memmove(unsafe.Pointer(p), noescape(unsafe.Pointer(&s)), 1)
			} else {
				*(*uint8)(unsafe.Pointer(p)) = uint8(2)
				*(*uint8)(unsafe.Pointer(p-1)) = uint8(10)
			}
		}
		span = spanOf(dst)
		if  span != nil && span.elemsize > 0 {
			idx   := span.objIndex(dst)
			x     := uintptr(idx)*span.elemsize+span.base()
			p     := (x + span.elemsize) - uintptr(1)
			lastp := uint8((span.limit-span.base())/span.elemsize)
			ref   := *(*uint8)(unsafe.Pointer(p))
			if ref > 0 && ref <= lastp {
				s := ref - uint8(1)
				if s >= 1 {
					memmove(unsafe.Pointer(p), noescape(unsafe.Pointer(&s)), 1)
					if s == 1 {
						*(*uint8)(unsafe.Pointer(p-1)) = uint8(111)
					}
				} else if s == 0 {
					if span.freelist != 0 {
						memmove(unsafe.Pointer(p-uintptr(1)), noescape(unsafe.Pointer(&span.freelist)), 2)
					} else {
						memmove(unsafe.Pointer(p-uintptr(1)), noescape(unsafe.Pointer(&s)), 2)
					}
					span.freelist = idx
				}
			} else {
				*(*uint8)(unsafe.Pointer(p)) = uint8(1)
				*(*uint8)(unsafe.Pointer(p-1)) = uint8(111)
			}
		}
	}
}

func tf(n uintptr) uint8 {
	return *(*uint8)(unsafe.Pointer(n))
}

func scanstatusanalyser(span *mspan) {
	for idx := span.objIndex(span.base()); idx < span.nelems; idx++ {
		p  := idx*span.elemsize + span.base()
		cc := tf(p - 1)
		rgb:= tf(p - 2)
		if cc == uint8(1) && rgb == uint8(111) {
			markred(p)
		}
		if rgb == uint8(100) && cc > 0 {
			scangreen(p)
		}
		if rgb == uint8(100) {
			collect(p)
		}
		if span.freelist != 0 {
			return
		}
	}
}

func collect(s uintptr) {
	x, _, _, span, sw := checkp(s)
	if sw == true {
		hbits := heapBitsForAddr(x)
		for i := uintptr(0); i < span.elemsize; i+=sys.PtrSize {
			if i != 1*sys.PtrSize && !hbits.morePointers() {
				break
			}
			if hbits.isPointer() {
				b := *(*uintptr)(unsafe.Pointer(x+i))
				next, _, _, _ := heapBitsForObject(b, x, i)
				_, _, nextrgb, _, nextsw := checkp(next)
				if nextsw == true {
					rcdelete(next)
					if tf(nextrgb) == uint8(100) {
						collect(next)
					}
				}
			}
			hbits = hbits.next()
		}
		*(*uint8)(unsafe.Pointer(x - 2)) = uint8(10)
		if span.freelist != 0 {
			memmove(unsafe.Pointer(x - 2), noescape(unsafe.Pointer(&span.freelist)), 2)
		}
		span.freelist = span.objIndex(x)
	}
}

func rcdelete(s uintptr) {
	x, _, _, span, sw := checkp(s)
	if sw == true {
		if tf(x - 1) == uint8(1) {
			hbits := heapBitsForAddr(x)
			for i := uintptr(0); i < span.elemsize; i+=sys.PtrSize {
				if i != 1*sys.PtrSize && !hbits.morePointers() {
					break
				}
				if hbits.isPointer() {
					b := *(*uintptr)(unsafe.Pointer(x+i))
					next, _, _, _ := heapBitsForObject(b, x, i)
					_, _, _, _, nextsw := checkp(next)
					if nextsw == true {
						rcdelete(next)
					}
				}
				hbits = hbits.next()
			}
			*(*uint8)(unsafe.Pointer(x - 2)) = (10)
			if span.freelist != 0 {
				memmove(unsafe.Pointer(x - 2), noescape(unsafe.Pointer(&span.freelist)), 2)
			}
			span.freelist = span.objIndex(x)
		} else {
			*(*uint8)(unsafe.Pointer(x - 1)) -= uint8(1)
			if tf(x - 2) != uint8(111){
				*(*uint8)(unsafe.Pointer(x - 2)) = uint8(111)
			}
		}
	}
}

func scangreen(s uintptr) {
// Green 10
	x, _, _, span, sw := checkp(s)
	if sw == true {
		*(*uint8)(unsafe.Pointer(x-2)) = uint8(10)
		hbits := heapBitsForAddr(x)
		for i := uintptr(0); i < span.elemsize; i+=sys.PtrSize {
			if i != 1*sys.PtrSize && !hbits.morePointers() {
				break
			}
			if hbits.isPointer() {
				b := *(*uintptr)(unsafe.Pointer(x+i))
				next, _, _, _ := heapBitsForObject(b, x, i)
				_, nextcc, nextrgb, _, nextsw := checkp(next)
				if nextsw == true {
					*(*uint8)(unsafe.Pointer(nextcc)) += uint8(1)
					if tf(nextrgb) != uint8(10) {
						scangreen(next)
					}
				}
			}
			hbits = hbits.next()
		}
	}
}

func checkp(p uintptr) (x, cc, rgb uintptr, span *mspan, sw bool) {
	sw = false
	if inheap(p) {
		span = spanOf(p)
		idx := span.objIndex(p)
		if idx <= span.nelems {
			x    = idx*span.elemsize+span.base()
			cc   = x - uintptr(1)
			rgb  = x - uintptr(2)
			sw = true
		}
	}
	return
}

func markred(obj uintptr) {
// Red       100
// B -> White 111
	x, _, rgb, span, sw := checkp(obj)
	if sw == true && tf(rgb) != uint8(100) {
		*(*uint8)(unsafe.Pointer(x-2)) = uint8(100)
		hbits := heapBitsForAddr(x)
		for i := uintptr(0); i < span.elemsize; i+=sys.PtrSize {
			if i != 1*sys.PtrSize && !hbits.morePointers() {
				break
			}
			if hbits.isPointer() {
				b := *(*uintptr)(unsafe.Pointer(x+i))
				next, _, _, _ := heapBitsForObject(b, x, i)
				_, nextcc, _, _, nextsw := checkp(next)
				if nextsw == true {
					*(*uint8)(unsafe.Pointer(nextcc)) -= uint8(1)
				}
			}
			hbits = hbits.next()
		}
		hbits = heapBitsForAddr(x)
		for i := uintptr(0); i < span.elemsize; i+=sys.PtrSize {
			if i != 1*sys.PtrSize && !hbits.morePointers() {
				break
			}
			if hbits.isPointer() {
				b := *(*uintptr)(unsafe.Pointer(x+i))
				next, _, _, _ := heapBitsForObject(b, x, i)
				_, nextcc, nextrgb, _, nextsw := checkp(next)
				if nextsw == true {
					if tf(nextrgb) != uint8(100) {
						markred(next)
					}
					if tf(nextcc) > 0 && tf(nextrgb) != uint8(111) {
						*(*uint8)(unsafe.Pointer(nextrgb)) = uint8(111)
					}
				}
			}
			hbits = hbits.next()
		}
	}
}
