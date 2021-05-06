package types

import "unsafe"
import "fmt"

/*
#cgo LDFLAGS: -L/opt/ros/foxy/lib -Wl,-rpath=/opt/ros/foxy/lib -lrcl -lrosidl_runtime_c -lrosidl_typesupport_cpp -lrosidl_typesupport_c -lstd_msgs__rosidl_generator_c -lstd_msgs__rosidl_typesupport_c
#cgo CFLAGS: -I/opt/ros/foxy/include
#include "std_msgs/msg/string.h"
#include "rosidl_runtime_c/string.h"
static inline const rosidl_message_type_support_t * ts_string(){
	const rosidl_message_type_support_t * ts =rosidl_typesupport_c__get_message_type_support_handle__std_msgs__msg__String();
    return ts;
}
*/
import "C"

//String ROS std message struct
type String struct {
	Data     *C.char
	Size     int
	Capacity int
}

//TypeSupport ROS msg typesupport
func (t *String) TypeSupport() unsafe.Pointer {
	fmt.Println("TypeSupport String called")
	return unsafe.Pointer(C.ts_string())
}

//GetData data getter
func (t *String) GetData() unsafe.Pointer {
	return unsafe.Pointer(t)
}

//Finish resource release
func (t *String) Finish() {
	C.free(unsafe.Pointer(t.Data))
}

//GenerateString generate String datatype fro string
func GenerateString(data string) *String {
	t := new(String)
	t.Data = C.CString(data)
	t.Size = len(data)
	t.Capacity = t.Size + 1
	return t
}

func (t *String) GetString() string {
	return C.GoString((*C.char)(t.Data))
}
