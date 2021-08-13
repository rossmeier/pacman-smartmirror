package apk

/*
#include <malloc.h>
#include "version.h"
*/
import "C"
import "unsafe"

func (*apkImpl) CompareVersions(a, b string) int {
	aC := C.CString(a)
	defer C.free(unsafe.Pointer(aC))
	bC := C.CString(b)
	defer C.free(unsafe.Pointer(bC))
	return int(C.apk_version_compare(aC, bC))
}
